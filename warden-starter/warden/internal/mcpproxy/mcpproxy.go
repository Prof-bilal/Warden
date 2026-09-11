// Package mcpproxy implements Warden's MCP-aware client proxy.
// This proxy sits between MCP clients (Claude Desktop, Cursor, etc.) and
// MCP servers (local or remote), providing filtering, auditing, and
// sensitive pattern blocking for MCP JSON-RPC communications.
package mcpproxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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
	TransportHTTP  MCPTransport = "http"
	TransportSSE   MCPTransport = "sse"
	TransportStdio MCPTransport = "stdio"
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
	Upstream      string   `yaml:"upstream"`       // "stdio:<command>", "https://host", or "http://loopback:P"
	AllowTools    []string `yaml:"allow_tools"`    // Allowed MCP tool names
	DenyPatterns  []string `yaml:"deny_patterns"`  // Regex patterns to block in payloads
	MaxPayloadKB  int      `yaml:"max_payload_kb"` // Max payload size in KB
	AuditRequests bool     `yaml:"audit_requests"` // Whether to log all requests
	EnvAllow      []string // Env var names passed to the stdio subprocess (deny-by-default)
}

// defaultHTTPTimeout bounds a single upstream HTTP request. MCP tool calls
// can legitimately be slow, but an upstream that never answers would pin a
// client connection forever; 10 minutes is generous for tool calls while
// still reaping dead upstreams.
const defaultHTTPTimeout = 10 * time.Minute

// ProxyServer is the MCP client proxy server
type ProxyServer struct {
	policy          MCPPolicy
	listener        net.Listener
	audit           *audit.Logger
	patterns        []*regexp.Regexp // Compiled deny patterns
	transport       MCPTransport
	upstream        string
	upstreamURL     *url.URL // parsed absolute URL when transport is http/sse
	client          *http.Client
	httpWriteMu     sync.Mutex // serializes client-connection writes + session id for http/sse transports
	upstreamSession string     // Mcp-Session-Id captured from the initialize response
}

// NewProxyServer creates a new MCP proxy server. All three transports are
// supported: stdio (subprocess bridge), http (Streamable HTTP), and sse
// (server-sent events). HTTP/SSE filtering is identical to stdio — every
// JSON-RPC message is checked before it is forwarded, and blocked messages
// are answered with a JSON-RPC error instead of reaching the upstream.
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

	if transport == TransportHTTP || transport == TransportSSE {
		u, err := url.Parse(upstream)
		if err != nil {
			return nil, fmt.Errorf("invalid mcp.upstream URL %q: %w", upstream, err)
		}
		if u.Scheme != "https" && u.Scheme != "http" {
			return nil, fmt.Errorf("mcp.upstream %q must be an http:// or https:// URL", upstream)
		}
		// Plain http:// is allowed only for loopback (local dev servers);
		// anything else would put MCP traffic — and any forwarded OAuth
		// tokens — on the wire unencrypted.
		if u.Scheme == "http" {
			host := u.Hostname()
			if host != "127.0.0.1" && host != "localhost" && host != "::1" && !net.ParseIP(host).IsLoopback() {
				return nil, fmt.Errorf("mcp.upstream %q uses plain http://; only loopback hosts may use http (use https:// for remote servers)", upstream)
			}
		}
		if u.Host == "" {
			return nil, fmt.Errorf("mcp.upstream %q has no host", upstream)
		}
		s.upstreamURL = u
		s.client = &http.Client{
			Timeout: defaultHTTPTimeout,
			// TLS verification is on by default; pinned explicitly so a future
			// refactor cannot silently weaken certificate checking.
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			},
		}
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

// handleConnection handles a single client connection, dispatching on the
// configured upstream transport: stdio bridges to a subprocess, HTTP/SSE
// speak newline-delimited JSON-RPC over the connection and re-issue each
// filtered message upstream over HTTP.
func (s *ProxyServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	switch s.transport {
	case TransportHTTP, TransportSSE:
		s.handleHTTPTransport(conn)
	default:
		s.handleStdioTransport(conn)
	}
}

// handleHTTPTransport bridges one client connection to a remote MCP server
// over HTTP. The client speaks the same newline-delimited JSON-RPC framing
// as the stdio transport; each line is filtered with the identical
// clientToChild pipeline, but "forwarding" means issuing an HTTP POST to
// the upstream instead of writing to a subprocess stdin pipe. The upstream
// response body (a JSON-RPC response, or for SSE the event stream it
// produces) is re-filtered line-by-line and relayed to the client.
//
// Fail-closed rules are the same as the stdio bridge: unparseable client
// lines get a JSON-RPC parse error, filtered messages never reach the
// upstream, and upstream output that is not JSON-RPC 2.0 is dropped and
// audited.
func (s *ProxyServer) handleHTTPTransport(conn net.Conn) {
	httpConn := conn
	clientErr := make(chan error, 2)
	upstreamDone := make(chan struct{})
	// sessCtx is canceled when the client-side loop exits, so the SSE
	// upstream reader stops promptly instead of pinning the upstream's
	// event stream after the client is gone.
	sessCtx, cancelSession := context.WithCancel(context.Background())
	defer cancelSession()

	// Client -> upstream: read JSON-RPC lines, filter, POST upstream.
	go func() {
		err := s.clientToUpstreamHTTP(sessCtx, conn)
		clientErr <- err
	}()

	// Upstream events -> client (SSE only). For plain HTTP the response to
	// each POST is relayed inline by clientToUpstreamHTTP, so no separate
	// reader is needed; for SSE the upstream pushes unsolicited events that
	// must be forwarded between requests.
	if s.transport == TransportSSE {
		go func() {
			s.upstreamSSEToClient(sessCtx, httpConn)
			close(upstreamDone)
		}()
	}

	err := <-clientErr
	// Stop the SSE reader before the deferred conn.Close: it may be blocked
	// reading the upstream, and its writes to conn must not race teardown.
	cancelSession()
	if s.transport == TransportSSE {
		<-upstreamDone
	}
	if err != nil && !isConnClosed(err) {
		s.logEvent("mcp_block", "client_read", false, fmt.Sprintf("client stream error: %v", err))
	}
}

// isConnClosed reports whether err is an expected connection teardown
// (client hung up, normal EOF) rather than a transport fault.
func isConnClosed(err error) bool {
	if err == nil || errors.Is(err, io.EOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "use of closed network connection") ||
		strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "broken pipe")
}

// clientToUpstreamHTTP reads newline-delimited JSON-RPC from the client,
// filters it, and forwards allowed requests to the HTTP upstream. Blocked
// or unparseable messages are answered with a JSON-RPC error to the client.
// For plain HTTP upstreams, the upstream's response is relayed back on the
// client connection in request order (MCP requests are sequential per
// connection here, which matches the stdio framing this listener speaks).
// The function returns when the client disconnects (EOF) or ctx is canceled.
func (s *ProxyServer) clientToUpstreamHTTP(ctx context.Context, conn net.Conn) error {
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 4*1024), s.scanCap())
	for sc.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var msg MCPMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil || msg.JSONRPC != "2.0" {
			s.logEvent("mcp_block", "unparseable_request", false, "client sent a non-JSON-RPC line; not forwarded")
			_ = s.writeClient(conn, &s.httpWriteMu, errorResponse(nil, -32700, "Parse error: message is not JSON-RPC 2.0 and was not forwarded by Warden"))
			continue
		}
		allowed, reason := s.filterMCPMessage(msg, "outbound")
		if !allowed {
			_ = s.writeClient(conn, &s.httpWriteMu, errorResponse(msg.ID, -32001, "blocked by Warden policy: "+reason))
			continue
		}
		if s.transport == TransportSSE {
			// SSE upstreams receive messages as POST bodies and answer inside
			// SSE-framed response bodies; relay those data lines to the client.
			if err := s.postSSEMessage(conn, line); err != nil {
				_ = s.writeClient(conn, &s.httpWriteMu, errorResponse(msg.ID, -32603, "upstream error: "+err.Error()))
			}
			continue
		}
		respBody, mediaType, err := s.postJSONRPC(line)
		if err != nil {
			_ = s.writeClient(conn, &s.httpWriteMu, errorResponse(msg.ID, -32603, "upstream error: "+err.Error()))
			continue
		}
		// Production MCP servers answer POSTs with SSE-framed bodies even on
		// the plain-http transport; dispatch on the declared media type and
		// keep raw JSON lines as the legacy path.
		if strings.Contains(mediaType, "text/event-stream") {
			s.relaySSE(conn, respBody)
		} else {
			s.relayUpstreamLines(conn, respBody)
		}
		respBody.Close()
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return nil
}

// relayUpstreamLines filters the upstream response body line-by-line and
// forwards allowed JSON-RPC 2.0 lines to the client. Non-JSON-RPC lines are
// dropped and audited — never forwarded.
func (s *ProxyServer) relayUpstreamLines(conn net.Conn, body io.Reader) {
	sc := bufio.NewScanner(body)
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
		if err := s.writeClient(conn, &s.httpWriteMu, line+"\n"); err != nil {
			return
		}
	}
}

// postJSONRPC sends one JSON-RPC message to the HTTP upstream as a POST and
// returns the response body plus its media type. Requests follow the
// Streamable HTTP convention: Accept: application/json, text/event-stream so
// servers may answer either way. When the server issued an Mcp-Session-Id at
// initialize time, it is replayed on every subsequent request.
func (s *ProxyServer) postJSONRPC(line string) (io.ReadCloser, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout())
	// Every early-return path must cancel; success hands ownership to the
	// returned ctxBody, whose Close() runs the cancel.
	fail := func(err error) (io.ReadCloser, string, error) {
		cancel()
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.upstreamURL.String(), bytes.NewBufferString(line+"\n"))
	if err != nil {
		return fail(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-06-18")
	if sid := s.sessionID(); sid != "" {
		req.Header.Set("Mcp-Session-Id", sid)
	}
	res, err := s.client.Do(req)
	if err != nil {
		return fail(err)
	}
	// A session-capable server hands out the id on the initialize response;
	// capture it exactly once per connection so later calls stay in-session.
	if sid := res.Header.Get("Mcp-Session-Id"); sid != "" {
		s.rememberSession(sid)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		io.Copy(io.Discard, io.LimitReader(res.Body, 1<<16))
		res.Body.Close()
		return fail(fmt.Errorf("upstream returned HTTP %d", res.StatusCode))
	}
	// Wrap so the context cancel fires when the caller drains the body.
	mediaType := res.Header.Get("Content-Type")
	return ctxBody{Reader: res.Body, cancel: cancel}, mediaType, nil
}

// sessionID returns the captured upstream Mcp-Session-Id, or "" before
// initialize has answered.
func (s *ProxyServer) sessionID() string {
	s.httpWriteMu.Lock()
	defer s.httpWriteMu.Unlock()
	return s.upstreamSession
}

// rememberSession stores the upstream Mcp-Session-Id exactly once: the id is
// minted at initialize and must not be overwritten by later responses.
func (s *ProxyServer) rememberSession(sid string) {
	s.httpWriteMu.Lock()
	defer s.httpWriteMu.Unlock()
	if s.upstreamSession == "" {
		s.upstreamSession = sid
	}
}

// ctxBody cancels the request context when the body is closed, releasing
// connection resources promptly.
type ctxBody struct {
	io.Reader
	cancel context.CancelFunc
}

func (b ctxBody) Close() error {
	if rc, ok := b.Reader.(io.Closer); ok {
		rc.Close()
	}
	b.cancel()
	return nil
}

// timeout returns the per-request timeout, shortened in tests.
func (s *ProxyServer) timeout() time.Duration {
	if s.client != nil && s.client.Timeout > 0 {
		return s.client.Timeout
	}
	return defaultHTTPTimeout
}

// postSSEMessage posts one JSON-RPC message to the SSE upstream and relays
// the SSE-framed response body to the client. Per the MCP Streamable HTTP
// transport, client-to-server messages ride POST request bodies and the
// server answers inside the response's event stream; those data payloads are
// the answer to the client's request and are filtered and forwarded like any
// other inbound message.
func (s *ProxyServer) postSSEMessage(conn net.Conn, line string) error {
	body, mediaType, err := s.postJSONRPC(line)
	if err != nil {
		return err
	}
	defer body.Close()
	if strings.Contains(mediaType, "text/event-stream") {
		s.relaySSE(conn, body)
	} else {
		s.relayUpstreamLines(conn, body)
	}
	return nil
}

// upstreamSSEToClient holds open the upstream's GET event stream (the
// server-to-client channel for SSE transport) and relays each event to the
// client as a JSON-RPC line, filtered like any other inbound message. It
// returns when the stream ends or sessCtx is canceled (client gone).
func (s *ProxyServer) upstreamSSEToClient(sessCtx context.Context, conn net.Conn) {
	ctx, cancel := context.WithCancel(sessCtx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.upstreamURL.String(), nil)
	if err != nil {
		s.logEvent("mcp_block", "sse_open", false, fmt.Sprintf("build GET: %v", err))
		return
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("MCP-Protocol-Version", "2025-06-18")
	if sid := s.sessionID(); sid != "" {
		req.Header.Set("Mcp-Session-Id", sid)
	}
	res, err := s.client.Do(req)
	if err != nil {
		s.logEvent("mcp_block", "sse_open", false, fmt.Sprintf("open SSE stream: %v", err))
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		// A server that does not offer a GET stream is spec-legal: responses
		// then arrive on the POST bodies. Only a 4xx/5xx is notable.
		if res.StatusCode >= 400 {
			s.logEvent("mcp_block", "sse_open", false, fmt.Sprintf("upstream SSE stream returned HTTP %d", res.StatusCode))
		}
		return
	}
	s.relaySSE(conn, res.Body)
}

// relaySSE reads an SSE body and forwards each event's data payload to the
// client connection, filtered like any other inbound message. The payload
// may itself span multiple `data:` lines per the SSE spec; every data line
// that decodes as JSON-RPC 2.0 is forwarded individually, matching the
// newline-delimited framing this listener speaks to its clients.
func (s *ProxyServer) relaySSE(conn net.Conn, body io.Reader) {
	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 4*1024), s.scanCap())
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, ":") || strings.HasPrefix(line, "event:") || strings.HasPrefix(line, "id:") || strings.HasPrefix(line, "retry:") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var msg MCPMessage
		if err := json.Unmarshal([]byte(payload), &msg); err != nil || msg.JSONRPC != "2.0" {
			s.logEvent("mcp_block", "unparseable_response", false, "SSE event was not JSON-RPC 2.0; dropped, not forwarded")
			continue
		}
		allowed, _ := s.filterMCPMessage(msg, "inbound")
		if !allowed {
			continue
		}
		if err := s.writeClient(conn, &s.httpWriteMu, payload+"\n"); err != nil {
			return
		}
	}
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
			_ = s.writeClient(conn, connMu, errorResponse(msg.ID, -32001, "blocked by Warden policy: "+reason))
			continue
		}
		if _, err := io.WriteString(childInW, line+"\n"); err != nil {
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
		if err := s.writeClient(conn, connMu, line+"\n"); err != nil {
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
	cap := 8 * 1024
	if s.policy.MaxPayloadKB > 0 {
		cap = s.policy.MaxPayloadKB*1024 + 1024
		if cap < 8*1024 {
			cap = 8 * 1024
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
	// The tool namespace is deny-by-default: everything else (initialize,
	// tools/list, resources/*, notifications) still passes so a client can
	// discover what it may call — enforcement happens at tools/call.
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
		Reason:   details,
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
