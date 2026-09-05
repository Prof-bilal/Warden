package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTempPolicy writes a policy YAML to a temp dir and returns its path.
func writeTempPolicy(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadValid(t *testing.T) {
	path := writeTempPolicy(t, `
command: ["node", "server.js"]
filesystem:
  read: ["./data"]
  write: ["./output"]
network:
  allow: ["api.github.com"]
env:
  allow: ["GITHUB_TOKEN"]
limits:
  memory_mb: 512
  timeout_s: 300
`)
	p, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	dir := filepath.Dir(path)
	if got, want := p.Filesystem.Read[0], filepath.Join(dir, "data"); got != want {
		t.Errorf("read[0] = %q, want %q", got, want)
	}
	if got, want := p.Filesystem.Write[0], filepath.Join(dir, "output"); got != want {
		t.Errorf("write[0] = %q, want %q", got, want)
	}
	if len(p.Command) != 2 || p.Command[0] != "node" {
		t.Errorf("Command = %v", p.Command)
	}
	if len(p.Network.Allow) != 1 || p.Network.Allow[0] != "api.github.com" {
		t.Errorf("Network.Allow = %v", p.Network.Allow)
	}
	if len(p.Env.Allow) != 1 || p.Env.Allow[0] != "GITHUB_TOKEN" {
		t.Errorf("Env.Allow = %v", p.Env.Allow)
	}
	if p.Limits.MemoryMB != 512 || p.Limits.TimeoutS != 300 {
		t.Errorf("Limits = %+v", p.Limits)
	}
}

func TestLoadRelativeResolvedAgainstPolicyDir(t *testing.T) {
	path := writeTempPolicy(t, `
filesystem:
  read: ["../shared", "./data"]
`)
	p, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	dir := filepath.Dir(path)
	want := []string{filepath.Join(dir, "..", "shared"), filepath.Join(dir, "data")}
	if p.Filesystem.Read[0] != want[0] || p.Filesystem.Read[1] != want[1] {
		t.Errorf("Read = %v, want %v", p.Filesystem.Read, want)
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	path := writeTempPolicy(t, `
command: ["true"]
filesystem:
  reed: ["./data"]   # typo: should be "read"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error for unknown field, got nil")
	}
	if !strings.Contains(err.Error(), "reed") {
		t.Errorf("error should name the bad field: %v", err)
	}
}

func TestLoadRejectsMalformedYAML(t *testing.T) {
	path := writeTempPolicy(t, "command: [unclosed\n")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestLoadRejectsEmptyGrantsOnly(t *testing.T) {
	// Empty read/write is legal (deny-by-default sandbox); the `command:`
	// absence is only an error once the CLI cannot supply it either.
	path := writeTempPolicy(t, `
filesystem: {}
`)
	p, err := Load(path)
	if err != nil {
		t.Fatalf("Load with no command should succeed (CLI may supply it): %v", err)
	}
	if err := p.ValidateRunnable(nil); err == nil {
		t.Fatal("expected error when neither policy nor CLI supplies a command")
	}
	if err := p.ValidateRunnable([]string{"true"}); err != nil {
		t.Fatalf("ValidateRunnable with CLI command: %v", err)
	}
}

func TestLoadValidWithEmptyFilesystem(t *testing.T) {
	path := writeTempPolicy(t, `
command: ["/bin/true"]
filesystem: {}
`)
	p, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(p.Filesystem.Read) != 0 || len(p.Filesystem.Write) != 0 {
		t.Errorf("expected empty grants, got %+v", p.Filesystem)
	}
}

func TestValidateRejectsBadHosts(t *testing.T) {
	for _, host := range []string{
		"",
		"api.github.com:443", // ports not allowed in schema
		"https://api.github.com",
		"api/github.com",
		"bad_host", // underscore not allowed in DNS hostnames
		"foo..com",
		"-leading.example",
		"trailing-.example",
	} {
		p := Policy{Command: []string{"true"}, Network: Network{Allow: []string{host}}}
		if err := p.Validate(); err == nil {
			t.Errorf("Validate: expected error for host %q, got nil", host)
		}
	}
}

func TestValidateAcceptsHostsAndIPs(t *testing.T) {
	for _, host := range []string{
		"api.github.com",
		"a-b.c-d.e",
		"localhost",
		"127.0.0.1",
		"192.168.1.1",
		"::1",
	} {
		p := Policy{Command: []string{"true"}, Network: Network{Allow: []string{host}}}
		if err := p.Validate(); err != nil {
			t.Errorf("Validate: unexpected error for host %q: %v", host, err)
		}
	}
}

func TestValidateRejectsRelativeCommandPath(t *testing.T) {
	p := Policy{Command: []string{"node", "server.js"}}
	if err := p.Validate(); err != nil {
		t.Fatalf("relative command is the policy author's concern, not a validation error: %v", err)
	}
}

func TestResolveCommand(t *testing.T) {
	p := Policy{Command: []string{"node", "server.js"}}

	cli := []string{"node", "server.js"}
	got, err := p.ResolveCommand(cli)
	if err != nil || len(got) != 2 {
		t.Fatalf("ResolveCommand(cli) = %v, %v", got, err)
	}

	got, err = p.ResolveCommand(nil)
	if err != nil || got[0] != "node" {
		t.Fatalf("ResolveCommand(policy) = %v, %v", got, err)
	}

	empty := Policy{}
	if _, err := empty.ResolveCommand(nil); err == nil {
		t.Fatal("expected error with neither CLI nor policy command")
	}
}

func TestEnvAllowlistDedupes(t *testing.T) {
	p := Policy{Env: Env{Allow: []string{"A", "B", "A", "C"}}}
	got := p.EnvAllowlist()
	want := []string{"A", "B", "C"}
	if len(got) != len(want) {
		t.Fatalf("EnvAllowlist() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("EnvAllowlist()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestValidateRejectsBadEnvNames(t *testing.T) {
	for _, name := range []string{"", "A B", "A=B", "A\tB"} {
		p := Policy{Command: []string{"true"}, Env: Env{Allow: []string{name}}}
		if err := p.Validate(); err == nil {
			t.Errorf("Validate: expected error for env name %q", name)
		}
	}
}

func TestUnimplementedAdvisories(t *testing.T) {
	p := Policy{Network: Network{Allow: []string{"api.github.com"}}, Limits: Limits{MemoryMB: 512}}
	advisories := p.UnimplementedAdvisories()
	if len(advisories) != 0 {
		t.Fatalf("UnimplementedAdvisories() = %v, want none", advisories)
	}

	none := Policy{}
	if got := none.UnimplementedAdvisories(); len(got) != 0 {
		t.Errorf("expected no advisories, got %v", got)
	}
}

func TestValidateRejectsNegativeLimits(t *testing.T) {
	for _, limits := range []Limits{{MemoryMB: -1}, {TimeoutS: -1}} {
		p := Policy{Limits: limits}
		if err := p.Validate(); err == nil {
			t.Errorf("Validate(%+v) succeeded, want error", limits)
		}
	}
}
