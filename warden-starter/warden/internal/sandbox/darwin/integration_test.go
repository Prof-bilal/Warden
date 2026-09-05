//go:build darwin

package darwin

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

func requireSandboxExec(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sandbox-exec"); err != nil {
		t.Skip("sandbox-exec not installed — skipping Seatbelt integration test")
	}
}

func TestSeatbeltBlocksUngrantedRead(t *testing.T) {
	requireSandboxExec(t)

	secretDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(secretDir, "secret.txt"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	scriptDir := t.TempDir()
	script := filepath.Join(scriptDir, "probe.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat \"$1\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Grant only the script dir — reading secretDir must fail.
	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{scriptDir}},
	}
	code, err := Run([]string{"/bin/sh", script, filepath.Join(secretDir, "secret.txt")}, p)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if code == 0 {
		t.Fatal("expected non-zero exit when reading ungranted path")
	}
}

func TestSeatbeltAllowsGrantedRead(t *testing.T) {
	requireSandboxExec(t)

	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "ok.txt"), []byte("allowed"), 0o644); err != nil {
		t.Fatal(err)
	}
	scriptDir := t.TempDir()
	script := filepath.Join(scriptDir, "probe.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat \"$1\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{dataDir, scriptDir}},
	}
	code, err := Run([]string{"/bin/sh", script, filepath.Join(dataDir, "ok.txt")}, p)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
}
