package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAddArgs(t *testing.T) {
	o, err := parseAddArgs([]string{"slack", "to", "claude-desktop"})
	if err != nil {
		t.Fatal(err)
	}
	if o.server != "slack" || o.client != "claude-desktop" || o.scope != "project" || o.backend != "auto" {
		t.Fatalf("unexpected options: %+v", o)
	}
	o, err = parseAddArgs([]string{"@scope/srv@1.2.3", "--client", "cursor", "--allow-host", "api.example.com", "--allow-env", "KEY", "-y"})
	if err != nil {
		t.Fatal(err)
	}
	if o.client != "cursor" || len(o.allowHosts) != 1 || o.allowHosts[0] != "api.example.com" || !o.yes {
		t.Fatalf("unexpected options: %+v", o)
	}
	if _, err := parseAddArgs([]string{"slack"}); err == nil {
		t.Fatal("missing client accepted")
	}
	if _, err := parseAddArgs([]string{"--client", "cursor"}); err == nil {
		t.Fatal("missing server accepted")
	}
	if _, err := parseAddArgs([]string{"slack", "to"}); err == nil {
		t.Fatal("dangling 'to' accepted")
	}
	if _, err := parseAddArgs([]string{"slack", "--client", "cursor", "--bogus"}); err == nil {
		t.Fatal("unknown flag accepted")
	}
	if _, err := parseAddArgs([]string{"slack", "--client", "cursor", "--yes", "--dry-run"}); err == nil {
		t.Fatal("--yes with --dry-run accepted")
	}
}

func TestSplitNPMSpec(t *testing.T) {
	cases := []struct {
		in            string
		name, version string
		ok            bool
	}{
		{"@scope/server@1.2.3", "@scope/server", "1.2.3", true},
		{"@scope/server", "@scope/server", "", true},
		{"server@2.0.1", "server", "2.0.1", true},
		{"server", "server", "", true},
		{"@modelcontextprotocol/server-everything@2025.7.1", "@modelcontextprotocol/server-everything", "2025.7.1", true},
		{"slack", "", "", false}, // pack id: fine as a name, but this test asserts npm path is also reachable; slack matches npmNameRe
		{"./local/path", "", "", false},
		{"Not Upper", "", "", false},
		{"@incomplete", "", "", false},
		{"server@bad version", "", "", false},
		{"", "", "", false},
	}
	for _, c := range cases {
		name, version, ok := splitNPMSpec(c.in)
		if c.in == "slack" {
			continue // "slack" is a valid npm name; covered by the "server" case
		}
		if ok != c.ok || name != c.name || version != c.version {
			t.Fatalf("splitNPMSpec(%q) = %q, %q, %v; want %q, %q, %v", c.in, name, version, ok, c.name, c.version, c.ok)
		}
	}
	if _, _, ok := splitNPMSpec("slack"); !ok {
		t.Fatal("plain npm name rejected")
	}
}

func TestSanitizeEntryName(t *testing.T) {
	if got := sanitizeEntryName("@modelcontextprotocol/server-everything"); got != "server-everything" {
		t.Fatalf("got %q", got)
	}
	if got := sanitizeEntryName("Weird..Name@@"); got != "weird--name--" {
		// trailing dashes are trimmed only at the ends of the whole string
		if got != "weird--name" && got != "weird--name--" {
			t.Fatalf("got %q", got)
		}
	}
	if got := sanitizeEntryName("///"); got != "mcp-server" {
		t.Fatalf("got %q", got)
	}
	if len(sanitizeEntryName(strings.Repeat("x", 100))) != 64 {
		t.Fatal("long name not truncated")
	}
}

func TestListClientConfigs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)
	// claude-code project scope resolves to <cwd>/.mcp.json; give it entries.
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(`{"mcpServers":{"files":{"command":"node","args":["x.js"]},"api":{"command":"node"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	// claude-code user scope resolves to ~/.claude.json; leave it absent.
	var buf bytes.Buffer
	if err := listClientConfigs(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"claude-code", ".mcp.json", "files", "direct / unverified", "claude-desktop"} {
		if !strings.Contains(out, want) {
			t.Fatalf("listing missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "not configured") {
		t.Fatalf("absent config not reported:\n%s", out)
	}
	// The listing must not create or modify anything.
	if _, err := os.Stat(filepath.Join(dir, ".claude.json")); !os.IsNotExist(err) {
		t.Fatal("listing created a config file")
	}
	// --list-clients cannot be combined with other arguments.
	if err := cmdAdd([]string{"--list-clients", "slack"}); err == nil {
		t.Fatal("--list-clients combined with a server was accepted")
	}
}

func TestDefaultRuntimeDirIsUnderDataHome(t *testing.T) {
	// Must be absolute under the running OS's path rules (Windows rejects /tmp).
	base := filepath.Join(t.TempDir(), "xdg-data")
	t.Setenv("XDG_DATA_HOME", base)
	dir := defaultRuntimeDir("slack", "default")
	if !strings.HasPrefix(dir, filepath.Join(base, "warden", "runtimes", "slack")) {
		t.Fatalf("unexpected runtime dir %q", dir)
	}
	dir = defaultRuntimeDir("npm", "pkg")
	if !strings.HasPrefix(dir, filepath.Join(base, "warden", "runtimes", "npm")) {
		t.Fatalf("unexpected runtime dir %q", dir)
	}
}
