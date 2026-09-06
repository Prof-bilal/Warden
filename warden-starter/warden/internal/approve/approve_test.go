package approve

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/proxy"
)

func TestParseNetAnswer(t *testing.T) {
	cases := []struct {
		in   string
		want proxy.Decision
	}{
		{"o", proxy.AllowOnce}, {"once", proxy.AllowOnce},
		{"s", proxy.AllowSession}, {"session", proxy.AllowSession},
		{"p", proxy.AllowAndSave}, {"save", proxy.AllowAndSave}, {"yes", proxy.AllowAndSave},
		{"", proxy.Deny}, {"d", proxy.Deny}, {"deny", proxy.Deny},
		{"nonsense", proxy.Deny}, {"allow everything forever", proxy.Deny},
	}
	for _, c := range cases {
		if got := parseNetAnswer(c.in); got != c.want {
			t.Errorf("parseNetAnswer(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseFileAnswer(t *testing.T) {
	cases := []struct {
		in   string
		want FileOutcome
	}{
		{"r", FileRestart}, {"restart", FileRestart},
		{"k", FileRecord}, {"record", FileRecord},
		{"", FileDeny}, {"d", FileDeny}, {"nonsense", FileDeny},
	}
	for _, c := range cases {
		if got := parseFileAnswer(c.in); got != c.want {
			t.Errorf("parseFileAnswer(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestShouldPromptFile(t *testing.T) {
	// Fixtures must live outside the sandbox runtime paths (/tmp is one),
	// so use this package's own directory, which exists on any checkout.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	pkgDir := filepath.Dir(thisFile)
	secret := filepath.Join(pkgDir, "approve.go")
	pol := policy.Policy{}

	ev := audit.Event{Type: "file", Action: "openat", Resource: secret, Allowed: false}
	grant, write, ok := ShouldPromptFile(ev, &pol, "/usr/bin/node")
	if !ok || grant != secret || write {
		t.Fatalf("ShouldPromptFile = %q, %v, %v; want %q, false, true", grant, write, ok, secret)
	}

	// Allowed events never prompt.
	ev.Allowed = true
	if _, _, ok := ShouldPromptFile(ev, &pol, "/usr/bin/node"); ok {
		t.Fatal("allowed event must not prompt")
	}
	ev.Allowed = false

	// Covered paths never prompt.
	pol.Filesystem.Read = []string{pkgDir}
	if _, _, ok := ShouldPromptFile(ev, &pol, "/usr/bin/node"); ok {
		t.Fatal("covered path must not prompt")
	}
	pol.Filesystem.Read = nil

	// Missing paths never prompt (routine probes, not denials worth asking).
	ev.Resource = filepath.Join(pkgDir, "does-not-exist.txt")
	if _, _, ok := ShouldPromptFile(ev, &pol, "/usr/bin/node"); ok {
		t.Fatal("missing path must not prompt")
	}

	// Runtime paths never prompt.
	ev.Resource = "/proc/self/cmdline"
	if _, _, ok := ShouldPromptFile(ev, &pol, "/usr/bin/node"); ok {
		t.Fatal("runtime path must not prompt")
	}

	// The server's own executable never prompts.
	ev.Resource = "/usr/bin/node"
	if _, _, ok := ShouldPromptFile(ev, &pol, "/usr/bin/node"); ok {
		t.Fatal("own executable must not prompt")
	}

	// Mutating actions propose a write grant on the parent dir.
	ev.Resource = filepath.Join(pkgDir, "newfile")
	ev.Action = "creat"
	grant, write, ok = ShouldPromptFile(ev, &pol, "/usr/bin/node")
	if !ok || grant != pkgDir || !write {
		t.Fatalf("creat = %q, %v, %v; want %q, true, true", grant, write, ok, pkgDir)
	}
}

func TestSaveGrantsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte("filesystem: {}\nnetwork: {}\nenv: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SaveHostGrant(path, "API.Example.COM"); err != nil {
		t.Fatalf("SaveHostGrant: %v", err)
	}
	dataDir := filepath.Join(dir, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveFileGrant(path, dataDir, false); err != nil {
		t.Fatalf("SaveFileGrant: %v", err)
	}
	p, err := policy.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(p.Network.Allow) != 1 || p.Network.Allow[0] != "api.example.com" {
		t.Errorf("hosts = %v", p.Network.Allow)
	}
	if len(p.Filesystem.Read) != 1 || p.Filesystem.Read[0] != dataDir {
		t.Errorf("read = %v", p.Filesystem.Read)
	}
	// A second identical save is a no-op success, not an error.
	if err := SaveHostGrant(path, "api.example.com"); err != nil {
		t.Fatalf("duplicate SaveHostGrant: %v", err)
	}
	if err := SaveFileGrant(path, "/usr/bin", true); err == nil {
		t.Fatal("write grant inside runtime base must fail")
	}
}

func TestConfigValidate(t *testing.T) {
	if err := (&Config{}).Validate(); err != nil {
		t.Fatalf("disabled config: %v", err)
	}
	if err := (&Config{Enabled: true}).Validate(); err == nil {
		t.Fatal("enabled without policy path must fail")
	}
	if err := (&Config{Enabled: true, PolicyPath: "p.yaml", Timeout: -1}).Validate(); err == nil {
		t.Fatal("negative timeout must fail")
	}
}
