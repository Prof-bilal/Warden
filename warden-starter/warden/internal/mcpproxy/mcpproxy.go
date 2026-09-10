// Package mcpproxy implements Warden's MCP-aware client proxy.
// This proxy sits between MCP clients (Claude Desktop, Cursor, etc.) and
// MCP servers (local or remote), providing filtering, auditing, and
// sensitive pattern blocking for MCP JSON-RPC communications.
package mcpproxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/proxy"
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
	Upstream      string            `yaml:"upstream"`       // "https://mcp.github.com" or "stdio:npx server"
	AllowTools    []string          `yaml:"allow_tools"`    // Allowed MCP tool names
	DenyPatterns  []string          `yaml:"deny_patterns"`  // Regex patterns to block in payloads
	AllowHosts    []string          `yaml:"allow_hosts"`    // Network hosts (for remote MCP)
	MaxPayloadKB  int              `yaml:"max_payload_kb"` // Max payload size in KB
	AuditRequests bool             `yaml:"audit_requests"` // Whether to log all requests
}

// ProxyServer is the MCP client proxy server
type ProxyServer struct {
	policy     MCPPolicy
	listener   net.Listener
	httpProxy  *proxy.Server  // Reuse existing HTTP proxy
	audit      *audit.Logger
	patterns   []*regexp.Regexp // Compiled deny patterns
	mu         sync.RWMutex
	transport  MCPTransport
	upstream   string
}

// NewProxyServer creates a new MCP proxy server
func NewProxyServer(policy MCPPolicy, logger *audit.Logger) (*ProxyServer, error) {
	// Parse upstream to determine transport
	transport, upstream, err := parseUpstream(policy.Upstream)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream %q: %w", policy.Upstream, err)
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

	// Create underlying HTTP proxy for remote MCP servers
	if transport == TransportHTTP || transport == TransportSSE {
		httpProxy, err := proxy.Start(policy.AllowHosts, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to start HTTP proxy: %w", err)
		}
		s.httpProxy = httpProxy
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

// handleConnection handles a single client connection
func (s *ProxyServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	switch s.transport {
	case TransportHTTP:
		s.handleHTTPTransport(conn)
	case TransportSSE:
		s.handleSSETransport(conn)
	case TransportStdio:
		s.handleStdioTransport(conn)
	default:
		s.logEvent("connection", "unknown_transport", false, fmt.Sprintf("unsupported transport: %s", s.transport))
	}
}

// handleHTTPTransport handles HTTP-based MCP communication
func (s *ProxyServer) handleHTTPTransport(conn net.Conn) {
	// Use existing HTTP proxy with MCP-specific filtering
	if s.httpProxy != nil {
		// Wrap the connection to intercept and filter MCP JSON-RPC
		wrapped := &mcpFilteringConn{
			Conn:   conn,
			server: s,
		}
		s.httpProxy.ServeHTTP(&wrappedResponseWriter{conn: wrapped}, &http.Request{})
	}
}

// handleSSETransport handles Server-Sent Events based MCP communication  
func (s *ProxyServer) handleSSETransport(conn net.Conn) {
	// SSE is similar to HTTP but with streaming JSON-RPC over event-stream
	s.handleHTTPTransport(conn) // Reuse HTTP handling for now
}

// handleStdioTransport handles stdio-based MCP communication (local servers)
func (s *ProxyServer) handleStdioTransport(conn net.Conn) {
	// For stdio transport, we need to proxy between the client connection
	// and a subprocess running the MCP server
	s.logEvent("stdio_proxy", s.upstream, true, "starting stdio MCP server")
	
	// TODO: Implement subprocess spawning and stdio bridging
	// This would involve:
	// 1. Parsing the stdio command from upstream (e.g., "stdio:npx @modelcontextprotocol/server-github")
	// 2. Starting the subprocess with controlled environment (using existing sandbox)
	// 3. Bridging JSON-RPC between client connection and subprocess stdio
	// 4. Applying MCP-specific filtering to the JSON-RPC messages
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

// Close shuts down the proxy server
func (s *ProxyServer) Close() error {
	var err error
	if s.listener != nil {
		err = s.listener.Close()
	}
	if s.httpProxy != nil {
		if err2 := s.httpProxy.Close(); err == nil {
			err = err2
		}
	}
	return err
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

// mcpFilteringConn wraps a connection to intercept and filter MCP messages
type mcpFilteringConn struct {
	net.Conn
	server *ProxyServer
	buffer bytes.Buffer
}

func (c *mcpFilteringConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if n > 0 {
		// Try to parse as MCP JSON-RPC and filter
		// This is a simplified implementation - real implementation would need
		// proper JSON-RPC message boundary detection
		c.buffer.Write(b[:n])
		if msg, valid := c.tryParseMCPMessage(); valid {
			allowed, reason := c.server.filterMCPMessage(msg, "inbound")
			if !allowed {
				c.server.logEvent("mcp_block", msg.Method, false, reason)
				return 0, fmt.Errorf("blocked MCP message: %s", reason)
			}
		}
	}
	return n, err
}

func (c *mcpFilteringConn) tryParseMCPMessage() (MCPMessage, bool) {
	// Try to parse JSON-RPC message from buffer
	// This is a simplified implementation
	var msg MCPMessage
	decoder := json.NewDecoder(&c.buffer)
	err := decoder.Decode(&msg)
	return msg, err == nil && msg.JSONRPC == "2.0"
}

// wrappedResponseWriter wraps an http.ResponseWriter to intercept responses
type wrappedResponseWriter struct {
	conn net.Conn
}

func (w *wrappedResponseWriter) Header() http.Header {
	return make(http.Header)
}

func (w *wrappedResponseWriter) Write(b []byte) (int, error) {
	return w.conn.Write(b)
}

func (w *wrappedResponseWriter) WriteHeader(statusCode int) {
	// Implementation would write HTTP status line
}

// Addr returns the listen address
func (s *ProxyServer) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}