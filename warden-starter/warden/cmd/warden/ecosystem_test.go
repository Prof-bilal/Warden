package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/clientconfig"
)

type ecosystemResult struct {
	stdout, stderr string
	exitCode       int
}

func runEcosystemTest(t *testing.T, bin string, args ...string) ecosystemResult {
	out, err, code := runWarden(t, bin, nil, args...)
	return ecosystemResult{out, err, code}
}

func TestClientPreviewDoesNotLaunchOrExposeSecrets(t *testing.T) {
	t.Setenv("WARDEN_SETUP_STATE_DIR", filepath.Join(t.TempDir(), "state"))
	bin := buildWarden(t)
	dir := t.TempDir()
	config := filepath.Join(dir, "mcp.json")
	policy := filepath.Join(dir, "policy.yaml")
	marker := filepath.Join(dir, "launched")
	configBytes, _ := json.Marshal(map[string]any{"mcpServers": map[string]any{"s": map[string]any{"command": "/bin/sh", "args": []string{"-c", "touch " + marker, "password-in-argv"}, "env": map[string]string{"TOKEN": "token-in-env"}}}})
	original := string(configBytes)
	if e := os.WriteFile(config, []byte(original), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(policy, []byte("filesystem: {}\n"), 0600); e != nil {
		t.Fatal(e)
	}
	result := runEcosystemTest(t, bin, "wrap", "--client", "generic", "--config", config, "--server", "s", "--policy", policy, "--dry-run")
	if result.exitCode != 0 {
		t.Fatalf("preview failed: %+v", result)
	}
	if strings.Contains(result.stdout+result.stderr, "password-in-argv") || strings.Contains(result.stdout+result.stderr, "token-in-env") {
		t.Fatal("preview exposed secret values")
	}
	got, _ := os.ReadFile(config)
	if string(got) != original {
		t.Fatal("preview changed config")
	}
	if _, e := os.Stat(marker); !os.IsNotExist(e) {
		t.Fatal("preview started target")
	}
	if _, e := os.Stat(filepath.Join(dir, ".warden")); !os.IsNotExist(e) {
		t.Fatal("preview wrote management files")
	}
	output := filepath.Join(dir, "export.json")
	result = runEcosystemTest(t, bin, "wrap", "--client", "generic", "--config", config, "--server", "s", "--policy", policy, "--output", output)
	if result.exitCode != 0 {
		t.Fatalf("export failed: %+v", result)
	}
	b, _ := os.ReadFile(output)
	var exported map[string]any
	if json.Unmarshal(b, &exported) != nil {
		t.Fatal("export malformed")
	}
	got, _ = os.ReadFile(config)
	if string(got) != original {
		t.Fatal("export mutated source")
	}
}

func TestEcosystemHelpAndCatalog(t *testing.T) {
	bin := buildWarden(t)
	for _, command := range []string{"packs", "clients", "wrap", "unwrap", "inventory"} {
		r := runEcosystemTest(t, bin, command, "--help")
		if r.exitCode != 0 || !strings.Contains(r.stderr, "Usage:") {
			t.Fatalf("help failed: %+v", r)
		}
	}
	r := runEcosystemTest(t, bin, "packs", "list", "--json")
	if r.exitCode != 0 {
		t.Fatal(r.stderr)
	}
	var packs []map[string]any
	if json.Unmarshal([]byte(r.stdout), &packs) != nil || len(packs) != 6 {
		t.Fatal("catalog output invalid")
	}
	for _, p := range packs {
		if p["status"] != "candidate" {
			t.Fatal("unverified profile promoted")
		}
	}
}

func TestPinnedPolicyFlagStopsAtCommandBoundary(t *testing.T) {
	if hasHelpFlag([]string{"--policy", "p", "--", "server", "--help"}) {
		t.Fatal("upstream help flag intercepted by Warden")
	}
	digest := strings.Repeat("a", 64)
	args := []string{"--policy", "p.yaml", "--policy-sha256", digest, "--", "server", "--policy-sha256", "upstream-argument"}
	got, rest, e := parsePolicyDigest(args)
	if e != nil || got != digest || !reflect.DeepEqual(rest, []string{"--policy", "p.yaml", "--", "server", "--policy-sha256", "upstream-argument"}) {
		t.Fatalf("pin parser changed upstream argv: %s %v %v", got, rest, e)
	}
	if _, _, e = parsePolicyDigest([]string{"--policy-sha256", "invalid"}); e == nil {
		t.Fatal("invalid pin accepted")
	}
}

func TestChangedPinnedPolicyRefusesBeforeLaunch(t *testing.T) {
	bin := buildWarden(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	_ = os.WriteFile(path, []byte("filesystem: {}"), 0600)
	result := runEcosystemTest(t, bin, "run", "--policy", path, "--policy-sha256", strings.Repeat("a", 64), "--", "missing-server")
	if result.exitCode == 0 || !strings.Contains(result.stderr, "policy changed since wrapping") {
		t.Fatalf("changed pin was not rejected before backend/launcher: %+v", result)
	}
}

// backupOverlapRefused reports whether wrap refused the policy because a
// grant overlaps the private backup storage. The production check has two
// phrasings: the direct path comparison and the symlink-resolved comparison
// (on macOS the temp root resolves /var -> /private/var, so the resolved
// branch fires first). Both are the same security refusal, so the assertion
// matches the shared "overlaps ... backup storage" reason rather than one
// branch's exact wording. The refusal itself is never weakened.
func backupOverlapRefused(r ecosystemResult) bool {
	return r.exitCode != 0 &&
		strings.Contains(r.stderr, "overlaps") &&
		strings.Contains(r.stderr, "backup storage")
}

func TestBackupStorageMustStayOutsideReadGrant(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WARDEN_SETUP_STATE_DIR", filepath.Join(dir, "private-state"))
	bin := buildWarden(t)
	config := filepath.Join(dir, "config.json")
	policy := filepath.Join(dir, "policy.yaml")
	_ = os.WriteFile(config, []byte(`{"mcpServers":{"s":{"command":"node","args":[]}}}`), 0600)
	content, _ := json.Marshal(dir)
	_ = os.WriteFile(policy, []byte("filesystem:\n  read: ["+string(content)+"]\n"), 0600)
	result := runEcosystemTest(t, bin, "wrap", "--client", "generic", "--config", config, "--server", "s", "--policy", policy, "--dry-run")
	if !backupOverlapRefused(result) {
		t.Fatalf("backup secret exposure not refused: %+v", result)
	}
	// A file inside private storage can expose saved launcher credentials even
	// when its parent directory is not granted. Refuse that overlap as well.
	storage := clientconfig.StateDirectory(config)
	content, _ = json.Marshal(filepath.Join(storage, "record.json"))
	_ = os.WriteFile(policy, []byte("filesystem:\n  read: ["+string(content)+"]\n"), 0600)
	result = runEcosystemTest(t, bin, "wrap", "--client", "generic", "--config", config, "--server", "s", "--policy", policy, "--dry-run")
	if !backupOverlapRefused(result) {
		t.Fatalf("file grant inside private storage was accepted: %+v", result)
	}
}
