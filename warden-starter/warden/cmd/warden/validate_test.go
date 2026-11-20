package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateUsesRuntimePolicyLoader(t *testing.T) {
	bin := buildWarden(t)
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.yaml")
	invalid := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(valid, []byte("filesystem: {read: [./data]}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalid, []byte("unknown_policy_field: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	env := []string{"XDG_STATE_HOME=" + dir, "NO_COLOR=1"}
	out, stderr, code := runWarden(t, bin, env, "validate", valid)
	if code != 0 || !strings.Contains(out, "Valid policy:") {
		t.Fatalf("validate valid policy exited %d, stdout %q, stderr %q", code, out, stderr)
	}
	out, stderr, code = runWarden(t, bin, env, "validate", valid, invalid)
	if code != 2 || !strings.Contains(out, "Valid policy:") || !strings.Contains(stderr, "unknown_policy_field") {
		t.Fatalf("validate mixed policies exited %d, stdout %q, stderr %q", code, out, stderr)
	}
	if err := os.WriteFile(valid, []byte("version: 1\nfilesystem: {read: [./data]}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	out, stderr, code = runWarden(t, bin, env, "validate", valid)
	if code != 0 || !strings.Contains(out, "Valid policy:") || !strings.Contains(stderr, "schema version 1 is deprecated") {
		t.Fatalf("validate legacy policy exited %d, stdout %q, stderr %q", code, out, stderr)
	}
}
