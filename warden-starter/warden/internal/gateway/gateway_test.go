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

func TestRejectUnsafeServerNames(t *testing.T) {
	for _, tc := range []struct {
		name, file, config string
	}{
		{"json traversal", "mcp.json", `{"mcpServers":{"../escape":{"command":"/bin/true"}}}`},
		{"json absolute separator", "mcp.json", `{"mcpServers":{"/tmp/escape":{"command":"/bin/true"}}}`},
		{"json backslash", "mcp.json", `{"mcpServers":{"..\\escape":{"command":"/bin/true"}}}`},
		{"json newline", "mcp.json", `{"mcpServers":{"bad\nname":{"command":"/bin/true"}}}`},
		{"windows drive", "mcp.json", `{"mcpServers":{"C:escape":{"command":"/bin/true"}}}`},
		{"reserved filename", "mcp.json", `{"mcpServers":{"..":{"command":"/bin/true"}}}`},
		{"yaml upstream traversal", "gateway.yaml", "upstreams:\n  - id: ../escape\n    command: /bin/true\n"},
		{"yaml backend traversal", "gateway.yaml", "backends:\n  ../escape:\n    command: /bin/true\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTemp(t, tc.file, tc.config)
			if _, err := Load(path); err == nil {
				t.Fatal("expected unsafe server name to be rejected")
			}
		})
	}
}

func TestWritePoliciesRejectsUnsafeName(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Servers: []ServerEntry{{Name: "../escape", Command: []string{"/bin/true"}}}}
	if _, _, err := WritePolicies(cfg, filepath.Join(dir, "policies")); err == nil {
		t.Fatal("expected unsafe server name to be rejected")
	}
	if _, err := os.Stat(filepath.Join(dir, "escape.yaml")); !os.IsNotExist(err) {
		t.Fatalf("policy escaped target dir: %v", err)
	}
}

func TestRejectDuplicateServerNames(t *testing.T) {
	for _, tc := range []struct {
		name, file, config string
	}{
		{"json aliases", "mcp.json", `{"mcpServers":{"same":{"type":"http","url":"https://example.com"}},"servers":{"same":{"command":"/bin/true"}}}`},
		{"json duplicate key", "mcp.json", `{"mcpServers":{"same":{"command":"/bin/true"},"same":{"command":"/bin/false"}}}`},
		{"yaml duplicate key", "gateway.yaml", "backends:\n  same:\n    command: /bin/true\n  same:\n    command: /bin/false\n"},
		{"yaml repeated upstream", "gateway.yaml", "upstreams:\n  - id: same\n    command: /bin/true\n  - id: same\n    command: /bin/false\n"},
		{"json trailing document", "mcp.json", `{"mcpServers":{"same":{"command":"/bin/true"}}}{"mcpServers":{"other":{"command":"/bin/false"}}}`},
		{"yaml registries", "gateway.yaml", "upstreams:\n  - id: same\n    command: /bin/true\nbackends:\n  same:\n    command: /bin/false\n"},
		{"yaml alias", "gateway.yaml", "upstreams:\n  - id: same\n    command: /bin/true\nmcpServers:\n  same:\n    command: /bin/false\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTemp(t, tc.file, tc.config)
			if _, err := Load(path); err == nil {
				t.Fatal("duplicate server name must fail closed")
			}
		})
	}
}

func TestLoadYAMLServersAlias(t *testing.T) {
	path := writeTemp(t, "gateway.yaml", "servers:\n  fs:\n    command: /bin/true\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	entry, err := cfg.Find("fs")
	if err != nil || entry.Remote || len(entry.Command) != 1 || entry.Command[0] != "/bin/true" {
		t.Fatalf("servers alias entry = %+v, %v", entry, err)
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
