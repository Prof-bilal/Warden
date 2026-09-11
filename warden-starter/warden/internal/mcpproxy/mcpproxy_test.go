package mcpproxy

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/warden-sandbox/warden/internal/audit"
)

// testProxy builds a ProxyServer over the stdio transport, so no Unix-socket
// HTTP proxy is started, and records audit events to a temp file.
func testProxy(t *testing.T, pol MCPPolicy) *ProxyServer {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(t.TempDir(), "audit.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	s, err := NewProxyServer(pol, audit.New(f))
	if err != nil {
		t.Fatalf("NewProxyServer: %v", err)
	}
	return s
}

func TestParseUpstream(t *testing.T) {
	cases := []struct {
		in        string
		wantType  MCPTransport
		wantUp    string
		wantError bool
	}{
		{"https://mcp.github.com/mcp", TransportHTTP, "https://mcp.github.com/mcp", false},
		{"https://mcp.stripe.com/stream", TransportSSE, "https://mcp.stripe.com/stream", false},
		{"stdio:npx @modelcontextprotocol/server-github", TransportStdio, "npx @modelcontextprotocol/server-github", false},
		{"", "", "", true},
		{"gopher://nope", "", "", true},
	}
	for _, c := range cases {
		tr, up, err := parseUpstream(c.in)
		if c.wantError {
			if err == nil {
				t.Errorf("parseUpstream(%q) expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseUpstream(%q): %v", c.in, err)
			continue
		}
		if tr != c.wantType || up != c.wantUp {
			t.Errorf("parseUpstream(%q) = (%q, %q), want (%q, %q)", c.in, tr, up, c.wantType, c.wantUp)
		}
	}
}

func TestFilterMCPMessageToolAllowlist(t *testing.T) {
	s := testProxy(t, MCPPolicy{
		Upstream:   "stdio:npx server",
		AllowTools: []string{"list_repos", "get_file"},
	})

	// Allowed tool call.
	msg := MCPMessage{JSONRPC: "2.0", Method: "tools/call", Params: map[string]interface{}{"name": "list_repos"}}
	allowed, reason := s.filterMCPMessage(msg, "outbound")
	if !allowed {
		t.Errorf("allowed tool blocked: %s", reason)
	}

	// Disallowed tool call must be denied by default.
	msg = MCPMessage{JSONRPC: "2.0", Method: "tools/call", Params: map[string]interface{}{"name": "admin_delete_all"}}
	allowed, reason = s.filterMCPMessage(msg, "outbound")
	if allowed {
		t.Error("disallowed tool was allowed")
	}
	if reason == "" {
		t.Error("expected a block reason for disallowed tool")
	}

	// Non tool-call methods (initialize, notifications) always pass.
	msg = MCPMessage{JSONRPC: "2.0", Method: "initialize", Params: map[string]interface{}{"protocolVersion": "2025-03-26"}}
	allowed, reason = s.filterMCPMessage(msg, "outbound")
	if !allowed {
		t.Errorf("initialize blocked: %s", reason)
	}
}

func TestFilterMCPMessageDenyPattern(t *testing.T) {
	// 36-char GitHub token, as used in the future.md schema sketch.
	s := testProxy(t, MCPPolicy{
		Upstream:     "stdio:npx server",
		DenyPatterns: []string{"ghp_[A-Za-z0-9]{36}"},
	})

	// Secret embedded in a payload must be blocked before egress.
	msg := MCPMessage{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  map[string]interface{}{"name": "create_issue", "arguments": map[string]interface{}{"notes": "token ghp_abcdefghijklmnopqrstuvwxyzABCDEF1234567"}},
	}
	allowed, reason := s.filterMCPMessage(msg, "outbound")
	if allowed {
		t.Error("message containing deny pattern was allowed")
	}
	if reason == "" {
		t.Error("expected a block reason for deny-pattern hit")
	}

	// Clean payload passes.
	msg = MCPMessage{JSONRPC: "2.0", Method: "tools/call", Params: map[string]interface{}{"name": "create_issue", "arguments": map[string]interface{}{"notes": "all good"}}}
	allowed, _ = s.filterMCPMessage(msg, "outbound")
	if !allowed {
		t.Error("clean message was blocked")
	}
}

func TestFilterMCPMessagePayloadLimit(t *testing.T) {
	s := testProxy(t, MCPPolicy{
		Upstream:     "stdio:npx server",
		MaxPayloadKB: 1,
	})

	big := map[string]interface{}{"name": "create_issue", "arguments": map[string]interface{}{"notes": string(make([]byte, 4096))}}
	msg := MCPMessage{JSONRPC: "2.0", Method: "tools/call", Params: big}
	allowed, _ := s.filterMCPMessage(msg, "outbound")
	if allowed {
		t.Error("oversized payload was allowed")
	}
}

func TestNewProxyServerInvalidDenyPattern(t *testing.T) {
	_, err := NewProxyServer(MCPPolicy{
		Upstream:     "stdio:npx server",
		DenyPatterns: []string{"["}, // invalid regex
	}, audit.New(nil))
	if err == nil {
		t.Fatal("expected error for invalid deny pattern")
	}
}

func TestNewProxyServerAcceptsHTTPUpstreams(t *testing.T) {
	// HTTP/SSE upstreams are now implemented: a server must be constructed,
	// and it must keep the parsed URL + client for the transport layer.
	s, err := NewProxyServer(MCPPolicy{Upstream: "https://mcp.github.com/mcp"}, audit.New(nil))
	if err != nil {
		t.Fatalf("https upstream rejected: %v", err)
	}
	if s.upstreamURL == nil || s.upstreamURL.Host != "mcp.github.com" {
		t.Fatalf("upstream URL not parsed, got %+v", s.upstreamURL)
	}
	if s.client == nil {
		t.Fatal("http client not configured")
	}
	if s.client.Timeout <= 0 {
		t.Fatal("http client must have a request timeout so a dead upstream cannot pin a client connection")
	}

	if _, err := NewProxyServer(MCPPolicy{Upstream: "https://mcp.stripe.com/stream"}, audit.New(nil)); err != nil {
		t.Fatalf("SSE upstream rejected: %v", err)
	}

	// stdio upstream: still accepted.
	if _, err := NewProxyServer(MCPPolicy{Upstream: "stdio:cat"}, audit.New(nil)); err != nil {
		t.Fatalf("stdio upstream rejected: %v", err)
	}
}

func TestNewProxyServerRejectsInsecureHTTPUpstreams(t *testing.T) {
	// Plain http:// is allowed only for loopback (local dev servers).
	// A remote plain-http upstream would put MCP traffic and any forwarded
	// tokens on the wire unencrypted — refuse it.
	_, err := NewProxyServer(MCPPolicy{Upstream: "http://mcp.example.com/mcp"}, audit.New(nil))
	if err == nil {
		t.Fatal("plain http:// to a remote host must be rejected")
	}
	if !strings.Contains(err.Error(), "https") {
		t.Fatalf("rejection should point at https, got: %v", err)
	}

	// Loopback http is fine (dev servers).
	for _, u := range []string{"http://127.0.0.1:8080/mcp", "http://localhost:8080/mcp"} {
		if _, err := NewProxyServer(MCPPolicy{Upstream: u}, audit.New(nil)); err != nil {
			t.Errorf("loopback upstream %q rejected: %v", u, err)
		}
	}

	// Non-http(s) schemes were never valid.
	if _, err := NewProxyServer(MCPPolicy{Upstream: "ftp://mcp.example.com"}, audit.New(nil)); err == nil {
		t.Error("ftp scheme must be rejected")
	}
}

// readLine reads one newline-delimited line from a client connection via a
// shared scanner, blocking until data arrives or the stream ends.
func readLine(sc *bufio.Scanner) string {
	if sc.Scan() {
		return strings.TrimSpace(sc.Text())
	}
	return ""
}

// waitAudit polls the JSONL audit file until it contains needle (the bridge
// writes events from goroutines, so the last block may land after the
// client-side assertion). Returns false on timeout.
func waitAudit(path, needle string) bool {
	for i := 0; i < 40; i++ {
		data, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(data), needle) {
			return true
		}
		<-time.After(50 * time.Millisecond)
	}
	return false
}

// TestStdioBridgeFiltersToolAllowlist is a real end-to-end test of the
// stdio transport: a TCP client, a real subprocess upstream (cat echoes the
// JSON-RPC line back), and the policy filter in between. It proves the
// bridge forwards allowed tool calls and blocks disallowed ones, and that
// both decisions reach the audit log.
func TestStdioBridgeFiltersToolAllowlist(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skipf("stdio bridge test requires a POSIX upstream (cat); filter logic is unit-tested on all platforms")
	}
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	f, err := os.OpenFile(auditPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	s, err := NewProxyServer(MCPPolicy{
		Upstream:   "stdio:cat",
		AllowTools: []string{"read_file"},
		EnvAllow:   []string{"PATH"},
	}, audit.New(f))
	if err != nil {
		t.Fatalf("NewProxyServer: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}

	conn, err := net.Dial("tcp", s.Addr())
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 4*1024), 1024*1024)

	// Allowed tool call: forwarded to the upstream, echoed back, allowed
	// inbound, and delivered to the client.
	allowed := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"read_file\"}}"
	if _, err := io.WriteString(conn, allowed+"\n"); err != nil {
		t.Fatalf("write allowed request: %v", err)
	}
	line := readLine(sc)
	if line == "" {
		t.Fatalf("no response for allowed tool call (is the bridge running?)")
	}
	if !strings.Contains(line, "\"id\":1") || !strings.Contains(line, "read_file") {
		t.Fatalf("allowed tool call not forwarded, got: %s", line)
	}

	// Disallowed tool call: answered with a JSON-RPC error, never forwarded.
	blocked := "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"delete_everything\"}}"
	if _, err := io.WriteString(conn, blocked+"\n"); err != nil {
		t.Fatalf("write blocked request: %v", err)
	}
	line = readLine(sc)
	if !strings.Contains(line, "\"error\"") {
		t.Fatalf("blocked tool call did not produce a JSON-RPC error, got: %s", line)
	}
	if !strings.Contains(line, "blocked by Warden policy") {
		t.Fatalf("blocked tool call error missing policy reason, got: %s", line)
	}

	// Clean shutdown: closing the client must terminate the upstream.
	conn.Close()

	f.Close()
	if !waitAudit(auditPath, "\"action\":\"mcp_block\"") {
		t.Fatalf("audit log has no mcp_block record; bridge decisions are not observable")
	}
	if !waitAudit(auditPath, "delete_everything") {
		t.Fatalf("audit log missing the blocked tool name")
	}
}

// TestStdioBridgeRejectsUnparseableRequest proves a non-JSON-RPC line from
// the client is never forwarded to the upstream: it gets a JSON-RPC parse
// error and an audit record, so the filter cannot be bypassed by breaking
// the framing.
func TestStdioBridgeRejectsUnparseableRequest(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skipf("stdio bridge test requires a POSIX upstream (cat)")
	}
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	f, err := os.OpenFile(auditPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	s, err := NewProxyServer(MCPPolicy{
		Upstream: "stdio:cat",
		EnvAllow: []string{"PATH"},
	}, audit.New(f))
	if err != nil {
		t.Fatalf("NewProxyServer: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}

	conn, err := net.Dial("tcp", s.Addr())
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 4*1024), 1024*1024)

	if _, err := io.WriteString(conn, "this is not json\n"); err != nil {
		t.Fatalf("write garbage: %v", err)
	}
	line := readLine(sc)
	if !strings.Contains(line, "\"code\":-32700") {
		t.Fatalf("expected a JSON-RPC parse error for garbage input, got: %s", line)
	}
	if line == "this is not json" {
		t.Fatalf("garbage must not be echoed back by the upstream")
	}

	conn.Close()
	f.Close()
	if !waitAudit(auditPath, "unparseable_request") {
		t.Fatalf("audit log missing the unparseable-request block")
	}
}

// startMCPServer spins up a local httptest server that answers JSON-RPC
// POSTs line-by-line and records the request bodies it received, so the
// tests can assert exactly what the proxy forwarded upstream.
func startMCPServer(t *testing.T, handler func(line string) (string, bool)) (*httptest.Server, *[]string) {
	t.Helper()
	var mu sync.Mutex
	var received []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body", http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, strings.TrimSpace(string(body)))
		mu.Unlock()
		// Production MCP servers answer POSTs with SSE-framed bodies
		// (text/event-stream with `data:` lines), which is what the live
		// interop test against mcp.deepwiki.com showed. Mirror that here so
		// the e2e tests pin the media-type dispatch the proxy performs.
		w.Header().Set("Content-Type", "text/event-stream")
		for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
			if line == "" {
				continue
			}
			resp, ok := handler(line)
			if !ok {
				continue
			}
			io.WriteString(w, "event: message\ndata: "+resp+"\n\n")
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &received
}

// echoHandler answers every tools/call with a JSON-RPC result naming the
// tool, and echoes other methods with a generic result.
func echoHandler(line string) (string, bool) {
	var req struct {
		ID     interface{} `json:"id"`
		Method string      `json:"method"`
		Params struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return "", false
	}
	result := map[string]interface{}{"echo": req.Method}
	if req.Method == "tools/call" {
		result = map[string]interface{}{"tool": req.Params.Name}
	}
	resp, _ := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": result})
	return string(resp), true
}

// dialProxy connects to a started proxy listener.
func dialProxy(t *testing.T, s *ProxyServer) (net.Conn, *bufio.Scanner) {
	t.Helper()
	conn, err := net.Dial("tcp", s.Addr())
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 4*1024), 1024*1024)
	return conn, sc
}

// TestHTTPTransportForwardsAllowedAndBlocksTool is the end-to-end proof of
// the HTTP transport: allowed tool calls are POSTed to the remote MCP server
// and the response is relayed; disallowed tool calls never reach the
// upstream and are answered with a JSON-RPC error; every decision is audited.
func TestHTTPTransportForwardsAllowedAndBlocksTool(t *testing.T) {
	srv, received := startMCPServer(t, echoHandler)
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	f, err := os.OpenFile(auditPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	s, err := NewProxyServer(MCPPolicy{
		Upstream:   srv.URL,
		AllowTools: []string{"read_file"},
	}, audit.New(f))
	if err != nil {
		t.Fatalf("NewProxyServer: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}

	conn, sc := dialProxy(t, s)

	// Allowed tool call: forwarded upstream, response relayed.
	allowed := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"read_file\"}}"
	if _, err := io.WriteString(conn, allowed+"\n"); err != nil {
		t.Fatalf("write allowed request: %v", err)
	}
	line := readLine(sc)
	if !strings.Contains(line, "\"tool\":\"read_file\"") {
		t.Fatalf("allowed tool response not relayed, got: %s", line)
	}

	// Disallowed tool call: JSON-RPC error, never forwarded.
	blocked := "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"delete_everything\"}}"
	if _, err := io.WriteString(conn, blocked+"\n"); err != nil {
		t.Fatalf("write blocked request: %v", err)
	}
	line = readLine(sc)
	if !strings.Contains(line, "\"error\"") || !strings.Contains(line, "blocked by Warden policy") {
		t.Fatalf("blocked tool call did not produce a policy error, got: %s", line)
	}

	conn.Close()
	for _, got := range *received {
		if strings.Contains(got, "delete_everything") {
			t.Fatalf("blocked tool call reached the upstream: %s", got)
		}
	}
	if len(*received) != 1 {
		t.Fatalf("upstream got %d requests, want exactly 1 (the allowed one): %v", len(*received), *received)
	}

	if !waitAudit(auditPath, "\"action\":\"mcp_block\"") {
		t.Fatal("audit log has no mcp_block record for the blocked HTTP request")
	}
}

// TestHTTPTransportDenyPatternBlocksBeforeEgress proves the deny-pattern
// filter fires on the outbound HTTP path: a payload containing a token-like
// string is stopped before it leaves loopback, and the upstream never sees it.
func TestHTTPTransportDenyPatternBlocksBeforeEgress(t *testing.T) {
	srv, received := startMCPServer(t, echoHandler)
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	f, err := os.OpenFile(auditPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	s, err := NewProxyServer(MCPPolicy{
		Upstream:     srv.URL,
		DenyPatterns: []string{"ghp_[A-Za-z0-9]{36}"},
	}, audit.New(f))
	if err != nil {
		t.Fatalf("NewProxyServer: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}

	conn, sc := dialProxy(t, s)

	secret := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"create_issue\",\"arguments\":{\"notes\":\"token ghp_abcdefghijklmnopqrstuvwxyzABCDEF1234567\"}}}"
	if _, err := io.WriteString(conn, secret+"\n"); err != nil {
		t.Fatalf("write secret request: %v", err)
	}
	line := readLine(sc)
	if !strings.Contains(line, "\"error\"") {
		t.Fatalf("deny-pattern payload was not blocked, got: %s", line)
	}

	clean := "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"create_issue\",\"arguments\":{\"notes\":\"all good\"}}}"
	if _, err := io.WriteString(conn, clean+"\n"); err != nil {
		t.Fatalf("write clean request: %v", err)
	}
	line = readLine(sc)
	if !strings.Contains(line, "\"tool\":\"create_issue\"") {
		t.Fatalf("clean request response missing, got: %s", line)
	}

	conn.Close()
	if len(*received) != 1 {
		t.Fatalf("upstream got %d requests, want exactly 1 (the clean one): %v", len(*received), *received)
	}
	for _, got := range *received {
		if strings.Contains(got, "ghp_") {
			t.Fatalf("secret payload reached the upstream: %s", got)
		}
	}
	if !waitAudit(auditPath, "blocked pattern") {
		t.Fatal("audit log missing the deny-pattern block")
	}
}

// TestHTTPTransportRejectsUnparseableAndBadJSONRPC mirrors the stdio bridge
// contract on the HTTP path: garbage in, JSON-RPC parse error out, nothing
// forwarded.
func TestHTTPTransportRejectsUnparseableAndBadJSONRPC(t *testing.T) {
	srv, received := startMCPServer(t, echoHandler)
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	f, err := os.OpenFile(auditPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	s, err := NewProxyServer(MCPPolicy{Upstream: srv.URL}, audit.New(f))
	if err != nil {
		t.Fatalf("NewProxyServer: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}

	conn, sc := dialProxy(t, s)

	if _, err := io.WriteString(conn, "this is not json\n"); err != nil {
		t.Fatalf("write garbage: %v", err)
	}
	line := readLine(sc)
	if !strings.Contains(line, "\"code\":-32700") {
		t.Fatalf("expected JSON-RPC parse error for garbage, got: %s", line)
	}

	// Valid JSON but not JSON-RPC 2.0: same treatment.
	if _, err := io.WriteString(conn, "{\"hello\":\"world\"}\n"); err != nil {
		t.Fatalf("write non-jsonrpc json: %v", err)
	}
	line = readLine(sc)
	if !strings.Contains(line, "\"code\":-32700") {
		t.Fatalf("expected parse error for non-2.0 json, got: %s", line)
	}

	conn.Close()
	if len(*received) != 0 {
		t.Fatalf("upstream got %d requests, want 0: %v", len(*received), *received)
	}
	if !waitAudit(auditPath, "unparseable_request") {
		t.Fatal("audit log missing the unparseable-request record")
	}
}

// TestHTTPTransportUpstreamOutageFailsClosed asserts the failure shape when
// the remote MCP server is down: the client gets a JSON-RPC internal error
// (-32603), never a silent hang or an unsandboxed fallback.
func TestHTTPTransportUpstreamOutageFailsClosed(t *testing.T) {
	// A server that is closed immediately: every POST fails at transport.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // free the port; connection will be refused

	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	f, err := os.OpenFile(auditPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	s, err := NewProxyServer(MCPPolicy{Upstream: srv.URL}, audit.New(f))
	if err != nil {
		t.Fatalf("NewProxyServer: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}

	conn, sc := dialProxy(t, s)
	req := "{\"jsonrpc\":\"2.0\",\"id\":7,\"method\":\"tools/call\",\"params\":{\"name\":\"read_file\"}}"
	if _, err := io.WriteString(conn, req+"\n"); err != nil {
		t.Fatalf("write request: %v", err)
	}
	line := readLine(sc)
	if !strings.Contains(line, "\"code\":-32603") || !strings.Contains(line, "upstream error") {
		t.Fatalf("upstream outage did not produce a JSON-RPC error, got: %s", line)
	}
	conn.Close()
}

// TestSSETransportRelaysServerEvents covers the SSE transport end to end:
// client POSTs are forwarded, the SSE-framed POST response is relayed to the
// client, and the server's unsolicited GET event stream is relayed too — all
// after the same inbound/outbound filtering as the stdio bridge.
func TestSSETransportRelaysServerEvents(t *testing.T) {
	var mu sync.Mutex
	var received []string

	// The server-to-client notification the GET stream will push once.
	notify, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0", "method": "notifications/changed",
		"params": map[string]string{"uri": "file:///x"},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			mu.Lock()
			received = append(received, strings.TrimSpace(string(body)))
			mu.Unlock()
			w.Header().Set("Content-Type", "text/event-stream")
			// Ack the POST with a JSON-RPC response inside an SSE body.
			var req struct {
				ID interface{} `json:"id"`
			}
			_ = json.Unmarshal(body, &req)
			resp, _ := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": map[string]string{"ok": "true"}})
			io.WriteString(w, "event: message\ndata: "+string(resp)+"\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case http.MethodGet:
			// Server-to-client event stream: push one notification, then hold
			// open until the client disconnects (context cancellation).
			w.Header().Set("Content-Type", "text/event-stream")
			io.WriteString(w, "data: "+string(notify)+"\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			<-r.Context().Done()
		}
	}))
	t.Cleanup(srv.Close)

	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	f, err := os.OpenFile(auditPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	s, err := NewProxyServer(MCPPolicy{Upstream: srv.URL + "/stream", AuditRequests: true}, audit.New(f))
	if err != nil {
		t.Fatalf("NewProxyServer: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}

	conn, sc := dialProxy(t, s)

	// First line: the notification relayed from the server's GET stream.
	deadline := time.Now().Add(5 * time.Second)
	var line string
	for time.Now().Before(deadline) {
		if conn.SetReadDeadline(deadline) == nil {
			line = readLine(sc)
			if line != "" {
				break
			}
		}
	}
	if !strings.Contains(line, "notifications/changed") {
		t.Fatalf("SSE server event not relayed to client, got: %q", line)
	}

	// A client request: POSTed upstream, the SSE-framed response relayed.
	req := "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"read_file\"}}"
	if _, err := io.WriteString(conn, req+"\n"); err != nil {
		t.Fatalf("write request: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	line = readLine(sc)
	if !strings.Contains(line, "\"ok\":\"true\"") {
		t.Fatalf("SSE POST response not relayed, got: %q", line)
	}

	conn.Close()
	if !waitAudit(auditPath, "mcp_message") {
		t.Fatal("audit log missing mcp_message record")
	}
}
