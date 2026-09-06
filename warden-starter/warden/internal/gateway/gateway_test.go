package gateway

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadClaudeJSON(t *testing.T) {
	p := writeTemp(t, "mcp.json", `{
  "mcpServers": {
    "fs": {"command": "/usr/bin/node", "args": ["server.js"], "env": {"TOKEN": "x"}},
    "web": {"type": "http", "url": "http://localhost:8000/mcp"}
  }
}`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Servers) != 2 {
		t.Fatalf("servers = %d, want 2", len(cfg.Servers))
	}
	fs, err := cfg.Find("fs")
	if err != nil || fs.Remote || len(fs.Command) != 2 {
		t.Fatalf("fs entry = %+v, %v", fs, err)
	}
	web, _ := cfg.Find("web")
	if !web.Remote {
		t.Fatalf("web should be remote: %+v", web)
	}
}

func TestLoadUpstreamsYAML(t *testing.T) {
	p := writeTemp(t, "gateway.yaml", `
upstreams:
  - id: "ctx"
    transport: "stdio"
    command: "npx"
    args: ["-y", "@upstash/context7-mcp"]
    env: {CTX_KEY: "v"}
  - id: "gh"
    transport: "streamable_http"
    endpoint: "https://api.githubcopilot.com/mcp/"
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ctx, err := cfg.Find("ctx")
	if err != nil || ctx.Remote {
		t.Fatalf("ctx = %+v, %v", ctx, err)
	}
	if ctx.Command[0] != "npx" || len(ctx.Command) != 3 {
		t.Fatalf("ctx.Command = %v", ctx.Command)
	}
	gh, _ := cfg.Find("gh")
	if !gh.Remote {
		t.Fatalf("gh should be remote")
	}
}

func TestLoadBackendsYAML(t *testing.T) {
	p := writeTemp(t, "gateway.yaml", `
backends:
  tavily:
    command: "npx -y @anthropic/mcp-server-tavily"
    env: {TAVILY_API_KEY: "${TAVILY_API_KEY}"}
  sentry:
    http_url: "https://mcp.sentry.dev/mcp"
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	tav, _ := cfg.Find("tavily")
	if tav.Remote || len(tav.Command) == 0 {
		t.Fatalf("tavily = %+v", tav)
	}
	sen, _ := cfg.Find("sentry")
	if !sen.Remote {
		t.Fatalf("sentry should be remote")
	}
}

func TestStarterPolicyDenyByDefault(t *testing.T) {
	e := ServerEntry{Name: "fs", Command: []string{"/usr/bin/node", "s.js"}, Env: map[string]string{"A": "1"}}
	pol, err := StarterPolicy(e)
	if err != nil {
		t.Fatalf("StarterPolicy: %v", err)
	}
	if len(pol.Filesystem.Read) != 0 || len(pol.Filesystem.Write) != 0 || len(pol.Network.Allow) != 0 {
		t.Fatalf("starter must be deny-by-default: %+v", pol)
	}
	if len(pol.Env.Allow) != 1 || pol.Env.Allow[0] != "A" {
		t.Fatalf("env.allow = %v", pol.Env.Allow)
	}
	if _, err := StarterPolicy(ServerEntry{Name: "r", Remote: true}); err == nil {
		t.Fatal("expected error for remote entry")
	}
}

func TestWritePoliciesNoOverwrite(t *testing.T) {
	p := writeTemp(t, "mcp.json", `{"mcpServers": {"a": {"command": "/bin/true"}}}`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "policies")
	written, skipped, err := WritePolicies(cfg, out)
	if err != nil || len(written) != 1 {
		t.Fatalf("WritePolicies = %v %v %v", written, skipped, err)
	}
	// Second call must not overwrite.
	written, skipped, err = WritePolicies(cfg, out)
	if err != nil || len(written) != 0 || len(skipped) != 1 {
		t.Fatalf("rewrite = %v %v %v", written, skipped, err)
	}
}

func TestWrapJSONKeepsEnv(t *testing.T) {
	p := writeTemp(t, "mcp.json", `{
  "mcpServers": {
    "a": {"command": "/bin/true", "args": ["x"], "env": {"K": "V"}},
    "r": {"type": "http", "url": "http://x/mcp"}
  }
}`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Wrap(cfg, "./policies", "/usr/bin/warden", "")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "warden") || !strings.Contains(s, "a.yaml") {
		t.Fatalf("wrapped missing warden prefix: %s", s)
	}
	if !strings.Contains(s, `"K"`) {
		t.Fatalf("wrapped must keep env values: %s", s)
	}
	if strings.Contains(s, "policies/r.yaml") {
		t.Fatalf("remote entry must not be wrapped: %s", s)
	}
}
