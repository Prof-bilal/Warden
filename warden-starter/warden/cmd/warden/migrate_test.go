package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/clientconfig"
	"github.com/warden-sandbox/warden/internal/policy"
)

func TestMigrateRepinsManagedLauncherAndPreservesUndo(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WARDEN_SETUP_STATE_DIR", filepath.Join(dir, "state"))
	policyPath := filepath.Join(dir, "warden.yaml")
	config := filepath.Join(dir, "client.json")
	originalPolicy := []byte("# comment\nversion: 1\nfilesystem:\n  read: [./shared]\n")
	originalConfig := []byte(`{"mcpServers":{"s":{"command":"server","args":[ "one", "two" ],"env":{"NAME":"private-test-value"}}}}` + "\n")
	writeMigrationFixture(t, policyPath, originalPolicy)
	writeMigrationFixture(t, config, originalConfig)
	plan, err := clientconfig.Prepare(config, "json", "s", policyPath, filepath.Join(dir, "warden"), "auto")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := clientconfig.Apply(plan); err != nil {
		t.Fatal(err)
	}
	wrapped := readMigrationFixture(t, config)
	args := []string{"--client", "generic", "--config", config, policyPath}
	assertMigrationRun(t, args, 3, "+ version: 2", "+ policy pin:")
	if _, err := os.Stat(policyPath + ".bak"); !os.IsNotExist(err) ||
		!bytes.Equal(readMigrationFixture(t, policyPath), originalPolicy) ||
		!bytes.Equal(readMigrationFixture(t, config), wrapped) {
		t.Fatal("preview wrote to disk")
	}
	assertMigrationRun(t, append([]string{"--write"}, args...), 0, "Migrated")
	upgraded := readMigrationFixture(t, policyPath)
	digest := clientconfig.Hash(upgraded)
	p, err := policy.LoadVerified(policyPath, digest)
	if err != nil || p.Version != 2 || len(p.Filesystem.Read) != 1 || p.Filesystem.Read[0] != filepath.Join(dir, "shared") ||
		!bytes.Contains(upgraded, []byte("# comment")) {
		t.Fatalf("policy migration changed the grants or lost its comment: %v, %v", p, err)
	}
	if !bytes.Equal(readMigrationFixture(t, policyPath+".bak"), originalPolicy) ||
		!bytes.Equal(readMigrationFixture(t, config+".bak"), wrapped) ||
		!bytes.Contains(readMigrationFixture(t, config), []byte(digest)) {
		t.Fatal("missing backup or updated launcher pin")
	}
	assertMigrationRun(t, args, 0)
	// Simulate interruption after replacing the policy and record, before the client config.
	writeMigrationFixture(t, config, wrapped)
	assertMigrationRun(t, append([]string{"--write"}, args...), 0, "verified against the policy backup")
	if undone, err := clientconfig.Undo(config, "s", false); err != nil || !undone {
		t.Fatalf("undo after repin = %v, %v", undone, err)
	}
	if !bytes.Equal(readMigrationFixture(t, config), originalConfig) {
		t.Fatal("undo did not restore the exact client bytes")
	}
}

func TestMigrateReportsLegacyProxyWithoutWidening(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "warden.yaml")
	config := filepath.Join(dir, "client.json")
	writeMigrationFixture(t, path, []byte("version: 1\nmcp: {upstream: 'stdio:fake-server'}\n"))
	raw, err := json.Marshal(map[string]any{"mcpServers": map[string]any{
		"legacy": map[string]any{"command": "warden", "args": []string{"proxy", "--policy", path}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	writeMigrationFixture(t, config, raw)
	assertMigrationRun(t, []string{"--write", "--client", "generic", "--config", config, path}, 3,
		"stdio proxy requires explicit --unsafe-unconfined", "empty mcp.allow_tools denies every tool", "Migrated")
	if !bytes.Equal(readMigrationFixture(t, config), raw) {
		t.Fatal("migration changed proxy launcher")
	}
	p, err := policy.Load(path)
	if err != nil || p.Version != 2 || len(p.MCP.AllowTools) != 0 {
		t.Fatalf("migration changed proxy tool grants: %v, %v", p, err)
	}
}

func TestMigrateRefusesStalePinAndUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "warden.yaml")
	config := filepath.Join(dir, "client.json")
	body := []byte("version: 1\nfilesystem: {}\n")
	writeMigrationFixture(t, path, body)
	raw, err := json.Marshal(map[string]any{"mcpServers": map[string]any{
		"stale": map[string]any{"command": "warden.exe", "args": []string{"run", "--policy", path, "--policy-sha256=" + strings.Repeat("1", 64)}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	writeMigrationFixture(t, config, raw)
	args := []string{"--write", "--client", "generic", "--config", config, path}
	assertMigrationRun(t, args, 3, "launcher digest does not match")
	if !bytes.Equal(readMigrationFixture(t, path), body) || !bytes.Equal(readMigrationFixture(t, config), raw) {
		t.Fatal("stale pin was silently repaired")
	}
	writeMigrationFixture(t, path, []byte("version: 1\nlegacy_key: private-test-value\n"))
	assertMigrationRun(t, args, 3, "field legacy_key not found")
	writeMigrationFixture(t, path, []byte("version: 1\n---\nfilesystem: {}\n"))
	assertMigrationRun(t, args, 3, "expected one YAML document")
}

func assertMigrationRun(t *testing.T, args []string, want int, messages ...string) {
	t.Helper()
	var output bytes.Buffer
	code, err := cmdMigrate(args, &output)
	if code != want || err != nil {
		t.Fatalf("migration = %d, %v; want %d\n%s", code, err, want, &output)
	}
	if strings.Contains(output.String(), "private-test-value") {
		t.Fatal("migration preview exposed a config value")
	}
	for _, message := range messages {
		if !strings.Contains(output.String(), message) {
			t.Fatalf("migration output missing %q:\n%s", message, &output)
		}
	}
}

func writeMigrationFixture(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func readMigrationFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
