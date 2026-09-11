// Package mcpproxy implements Warden's MCP-aware client proxy.
// This proxy sits between MCP clients (Claude Desktop, Cursor, etc.) and
// MCP servers (local or remote), providing filtering, auditing, and
// sensitive pattern blocking for MCP JSON-RPC communications.
package mcpproxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/envfilter"
)

// MCPTransport represents the transport type for MCP communication
type MCPTransport string

const (
	TransportHTTP   MCPTransport = "http"
	TransportSSE    MCPTransport = "sse"
	TransportStdio  MCPTransport = "stdio"
)

// MCPMessage represents a JSON-RPC message in MCP protocol
type MCPMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// MCPPolicy extends the standard policy with MCP-specific configuration
type MCPPolicy struct {
	Upstream      string            `yaml:"upstream"`       // "stdio:<command>" (HTTP/SSE rejected as not implemented)
	AllowTools    []string          `yaml:"allow_tools"`    // Allowed MCP tool names
	DenyPatterns  []string          `yaml:"deny_patterns"`  // Regex patterns to block in payloads
	MaxPayloadKB  int              `yaml:"max_payload_kb"` // Max payload size in KB
	AuditRequests bool             `yaml:"audit_requests"` // Whether to log all requests
	EnvAllow      []string          // Env var names passed to the stdio subprocess (deny-by-default)
}

// ProxyServer is the MCP client proxy server
type ProxyServer struct {
	policy     MCPPolicy
	listener   net.Listener
	audit      *audit.Logger
	patterns   []*regexp.Regexp // Compiled deny patterns
	transport  MCPTransport
	upstream   string
}

// NewProxyServer creates a new MCP proxy server. Only the stdio transport
// is implemented; HTTP and SSE upstreams are refused at construction so a
// user never gets a listener that pretends to enforce a policy it cannot.
func NewProxyServer(policy MCPPolicy, logger *audit.Logger) (*ProxyServer, error) {
	// Parse upstream to determine transport
	transport, upstream, err := parseUpstream(policy.Upstream)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream %q: %w", policy.Upstream, err)
	}
	if transport != TransportStdio {
		return nil, fmt.Errorf("unsupported mcp.upstream transport %q: only \"stdio:<command>\" is implemented (HTTP/SSE upstreams are not supported yet); use a stdio upstream such as \"stdio:npx @modelcontextprotocol/server-github\"", policy.Upstream)
	}

	s := &ProxyServer{
		policy:    policy,
		audit:     logger,
		transport: transport,
		upstream:  upstream,
	}

	// Compile deny patterns
	for _, pattern := range policy.DenyPatterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid deny pattern %q: %w", pattern, err)
		}
		s.patterns = append(s.patterns, regex)
	}

	return s, nil
}

// Start starts the MCP proxy server
func (s *ProxyServer) Start(listen string) error {
	listener, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", listen, err)
	}
	s.listener = listener

	s.logEvent("proxy_start", listen, true, fmt.Sprintf("MCP proxy started, transport=%s", s.transport))

	go s.serve()
	return nil
}

// serve handles incoming client connections
func (s *ProxyServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single client connection. Only the stdio
// transport is implemented (NewProxyServer rejects the others at startup),
// so every connection is bridged to a local stdio MCP server subprocess.
func (s *ProxyServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	s.handleStdioTransport(conn)
}

// handleStdioTransport bridges one client connection to a local stdio MCP
// server subprocess, filtering every JSON-RPC message in both directions.
// Framing is newline-delimited JSON-RPC 2.0 (one message per line), which is
// what the MCP SDK's stdio transport uses.
//
// Fail-closed rules:
//   - a client line that is not valid JSON-RPC 2.0 is never forwarded: it is
//     audited and answered with a JSON-RPC parse error;
//   - a filtered request (tool not allowed / deny pattern / too large) is
//     answered with a JSON-RPC error and never reaches the subprocess;
//   - a subprocess line that is not valid JSON-RPC 2.0 is dropped and
//     audited — never forwarded to the client;
//   - a filtered response is dropped and audited so sensitive data cannot
//     leak to the client.
func (s *ProxyServer) handleStdioTransport(conn net.Conn) {
	argv := strings.Fields(s.upstream)
	if len(argv) == 0 {
		s.logEvent("mcp_block", "stdio", false, "empty stdio upstream command")
		return
	}

	childInR, childInW, err := os.Pipe()
	if err != nil {
		s.logEvent("mcp_block", "stdio_pipe", false, fmt.Sprintf("create stdin pipe: %v", err))
		return
	}
	defer childInR.Close()
	defer childInW.Close()
	childOutR, childOutW, err := os.Pipe()
	if err != nil {
		s.logEvent("mcp_block", "stdio_pipe", false, fmt.Sprintf("create stdout pipe: %v", err))
		return
	}
	defer childOutR.Close()
	defer childOutW.Close()

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = childInR
	cmd.Stdout = childOutW
	cmd.Stderr = os.Stderr
	// The subprocess inherits only env.allow names (deny-by-default).
	cmd.Env = envfilter.Filter(os.Environ(), s.policy.EnvAllow)

	if err := cmd.Start(); err != nil {
		s.logEvent("mcp_block", argv[0], false, fmt.Sprintf("start stdio upstream: %v", err))
		return
	}
	s.logEvent("stdio_proxy", argv[0], true, "stdio MCP upstream started")

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	connMu := sync.Mutex{}
	clientClosed := make(chan struct{}, 2)
	bothDone := make(chan struct{})
	go func() {
		<-clientClosed
		<-clientClosed
		bothDone <- struct{}{}
	}()

	// Client -> subprocess.
	go func() {
		// When the client goes away, close the write end so the subprocess
		// sees EOF on stdin (and can exit on its own).
		defer func() { _ = childInW.Close() }()
		s.clientToChild(conn, childInW, &connMu)
		clientClosed <- struct{}{}
	}()
	// Subprocess -> client.
	go func() {
		defer func() { _ = conn.Close() }()
		s.childToClient(childOutR, conn, &connMu)
		clientClosed <- struct{}{}
	}()

	select {
	case <-done:
		// The subprocess exited on its own: drop the client connection and
		// let the direction goroutines unwind (they will see EOF/errors).
		_ = conn.Close()
		_ = childInW.Close()
	case <-bothDone:
		// Both bridge directions finished (client gone / upstream stdout
		// closed): stop the subprocess if it is still running, then reap it.
		terminateChild(cmd, done)
	}
}

// clientToChild reads newline-delimited JSON-RPC from the client, filters
// it, and forwards allowed requests to the subprocess stdin. Blocked or
// unparseable messages are answered with a JSON-RPC error to the client.
func (s *ProxyServer) clientToChild(conn net.Conn, childInW io.Writer, connMu *sync.Mutex) {
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 4*1024), s.scanCap())
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var msg MCPMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil || msg.JSONRPC != "2.0" {
			s.logEvent("mcp_block", "unparseable_request", false, "client sent a non-JSON-RPC line; not forwarded")
			_ = s.writeClient(conn, connMu, errorResponse(nil, -32700, "Parse error: message is not JSON-RPC 2.0 and was not forwarded by Warden"))
			continue
		}
		allowed, reason := s.filterMCPMessage(msg, "outbound")
		if !allowed {
			_ = s.writeClient(conn, connMu, errorResponse(msg.ID, -32001, "blocked by Warden policy: " + reason))
			continue
		}
		if _, err := io.WriteString(childInW, line + "\n"); err != nil {
			return
		}
	}
	if err := sc.Err(); err != nil {
		s.logEvent("mcp_block", "client_read", false, fmt.Sprintf("client stream error: %v", err))
	}
}

// childToClient reads newline-delimited JSON-RPC responses from the
// subprocess, filters them, and forwards allowed responses to the client.
// Filtered responses (deny pattern / oversize) are dropped, never forwarded.
func (s *ProxyServer) childToClient(childOutR io.Reader, conn net.Conn, connMu *sync.Mutex) {
	sc := bufio.NewScanner(childOutR)
	sc.Buffer(make([]byte, 4*1024), s.scanCap())
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var msg MCPMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil || msg.JSONRPC != "2.0" {
			s.logEvent("mcp_block", "unparseable_response", false, "upstream sent a non-JSON-RPC line; dropped, not forwarded")
			continue
		}
		allowed, _ := s.filterMCPMessage(msg, "inbound")
		if !allowed {
			continue
		}
		if err := s.writeClient(conn, connMu, line + "\n"); err != nil {
			return
		}
	}
	if err := sc.Err(); err != nil {
		s.logEvent("mcp_block", "upstream_read", false, fmt.Sprintf("upstream stream error: %v", err))
	}
}

// writeClient serializes writes to the client connection so the two bridge
// goroutines (blocked-request errors and forwarded responses) never
// interleave partial lines.
func (s *ProxyServer) writeClient(conn net.Conn, mu *sync.Mutex, line string) error {
	mu.Lock()
	defer mu.Unlock()
	_, err := io.WriteString(conn, line)
	return err
}

// errorResponse renders a JSON-RPC 2.0 error reply.
func errorResponse(id interface{}, code int, message string) string {
	resp := MCPMessage{JSONRPC: "2.0", ID: id, Error: map[string]interface{}{"code": code, "message": message}}
	data, err := json.Marshal(resp)
	if err != nil {
		return "{\"jsonrpc\":\"2.0\",\"id\":null,\"error\":{\"code\":-32603,\"message\":\"internal error: encode reply\"}}\n"
	}
	return string(data) + "\n"
}

// scanCap bounds the per-line read buffer: max_payload_kb when set, else an
// 8 MiB safety cap so an oversized line cannot exhaust memory.
func (s *ProxyServer) scanCap() int {
	cap := 8*1024
	if s.policy.MaxPayloadKB > 0 {
		cap = s.policy.MaxPayloadKB*1024 + 1024
		if cap < 8*1024 {
			cap = 8*1024
		}
	}
	return cap
}

// terminateChild signals the subprocess to stop, escalates to SIGKILL after
// a short grace period, and reaps it via done.
func terminateChild(cmd *exec.Cmd, done <-chan error) {
	_ = cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-done:
		return
	case <-time.After(750 * time.Millisecond):
		_ = cmd.Process.Signal(syscall.SIGKILL)
		_ = <-done
	}
}

// filterMCPMessage examines an MCP message and decides whether to allow it.
// Denied messages are audited as mcp_block so the evidence trail exists even
// when the caller forgets to log the block itself.
func (s *ProxyServer) filterMCPMessage(msg MCPMessage, direction string) (bool, string) {
	// Log all requests if auditing is enabled
	if s.policy.AuditRequests {
		s.logEvent("mcp_message", msg.Method, true, fmt.Sprintf("direction=%s id=%v", direction, msg.ID))
	}

	reason := s.blockReason(msg)
	if reason == "" {
		return true, ""
	}
	s.logEvent("mcp_block", msg.Method, false, reason)
	return false, reason
}

// blockReason returns why msg must be blocked, or "" if it may pass. The
// tool allowlist, deny patterns, and payload limit checks are all evaluated
// here so deny-by-default is enforced even for messages with empty fields.
func (s *ProxyServer) blockReason(msg MCPMessage) string {
	// Check tool allowlist for tool calls. Per the MCP spec the invoked
	// tool name lives in params.name, not in the JSON-RPC method.
	if msg.Method == "tools/call" {
		toolName := toolNameFromParams(msg.Params)
		if !s.isToolAllowed(toolName) {
			return fmt.Sprintf("tool %q not in allow list", toolName)
		}
	}

	// Check for sensitive patterns in the message content
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return "failed to serialize message for pattern checking"
	}

	for _, pattern := range s.patterns {
		if pattern.Match(msgBytes) {
			return fmt.Sprintf("message contains blocked pattern: %s", pattern.String())
		}
	}

	// Check payload size limits
	if s.policy.MaxPayloadKB > 0 {
		sizeKB := len(msgBytes) / 1024
		if sizeKB > s.policy.MaxPayloadKB {
			return fmt.Sprintf("payload size %dKB exceeds limit %dKB", sizeKB, s.policy.MaxPayloadKB)
		}
	}

	return ""
}

// toolNameFromParams extracts the invoked tool name from a tools/call
// params object. A non-map or missing name yields "" (deny-by-default).
func toolNameFromParams(params interface{}) string {
	m, ok := params.(map[string]interface{})
	if !ok {
		return ""
	}
	name, _ := m["name"].(string)
	return name
}

// isToolAllowed checks if a tool name is in the allowlist
func (s *ProxyServer) isToolAllowed(toolName string) bool {
	if len(s.policy.AllowTools) == 0 {
		return true // No restrictions if allowlist is empty
	}

	for _, allowed := range s.policy.AllowTools {
		if allowed == toolName || allowed == "*" {
			return true
		}
	}
	return false
}

// logEvent logs an audit event. A nil logger (e.g. during tests that only
// exercise filtering) is a no-op rather than a crash.
func (s *ProxyServer) logEvent(eventType, resource string, allowed bool, details string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Log(audit.Event{
		Type:     "mcp_proxy",
		Action:   eventType,
		Resource: resource,
		Allowed:  allowed,
		Reason:    details,
	})
}

// Close shuts down the proxy server.
func (s *ProxyServer) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// parseUpstream parses the upstream configuration to determine transport and target
func parseUpstream(upstream string) (MCPTransport, string, error) {
	if strings.HasPrefix(upstream, "stdio:") {
		return TransportStdio, strings.TrimPrefix(upstream, "stdio:"), nil
	}
	if strings.HasPrefix(upstream, "http://") || strings.HasPrefix(upstream, "https://") {
		// Determine if it's SSE based on common patterns
		if strings.Contains(upstream, "stream") || strings.Contains(upstream, "events") {
			return TransportSSE, upstream, nil
		}
		return TransportHTTP, upstream, nil
	}
	if upstream == "" {
		return "", "", fmt.Errorf("upstream cannot be empty")
	}
	return "", "", fmt.Errorf("unsupported upstream format: %s", upstream)
}

// Addr returns the listen address
func (s *ProxyServer) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}