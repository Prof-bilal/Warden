package mcpproxy

import (
	"os"
	"path/filepath"
	"testing"

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
