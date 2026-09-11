package mcpproxy

import (
	"bufio"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestNewProxyServerRejectsUnsupportedTransports(t *testing.T) {
	// HTTP upstream: the transport is not implemented — refusing at
	// construction avoids a listener that pretends to enforce.
	_, err := NewProxyServer(MCPPolicy{Upstream: "https://mcp.github.com/mcp"}, audit.New(nil))
	if err == nil {
		t.Fatalf("expected HTTP upstream to be rejected")
	}
	if !strings.Contains(err.Error(), "stdio") {
		t.Fatalf("rejection should point to the supported stdio transport, got: %v", err)
	}

	// SSE upstream: same fail-closed treatment.
	_, err = NewProxyServer(MCPPolicy{Upstream: "https://mcp.stripe.com/stream"}, audit.New(nil))
	if err == nil {
		t.Fatalf("expected SSE upstream to be rejected")
	}

	// stdio upstream: accepted.
	if _, err := NewProxyServer(MCPPolicy{Upstream: "stdio:cat"}, audit.New(nil)); err != nil {
		t.Fatalf("stdio upstream rejected: %v", err)
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
	if _, err := io.WriteString(conn, allowed + "\n"); err != nil {
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
	if _, err := io.WriteString(conn, blocked + "\n"); err != nil {
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