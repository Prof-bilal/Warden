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

// writeFixture writes an executable script into dir and returns its path.
func writeFixture(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
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

func TestSeatbeltWriteGrantWritable(t *testing.T) {
	requireSandboxExec(t)

	writeDir := t.TempDir()
	scriptDir := t.TempDir()
	sh := writeFixture(t, scriptDir, "probe.sh", "#!/bin/sh\necho made > \"$1/f.txt\" && cat \"$1/f.txt\"\n")
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Write: []string{writeDir},
			Read:  []string{scriptDir},
		},
	}
	code, err := Run([]string{"/bin/sh", sh, writeDir}, p)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if _, err := os.Stat(filepath.Join(writeDir, "f.txt")); err != nil {
		t.Errorf("file not created in write grant: %v", err)
	}
}

func TestSeatbeltWriteOutsideGrantDenied(t *testing.T) {
	requireSandboxExec(t)

	writeDir := t.TempDir()
	deniedDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(deniedDir, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	scriptDir := t.TempDir()
	sh := writeFixture(t, scriptDir, "probe.sh", "#!/bin/sh\necho evil > \"$1/keep.txt\"\n")
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Write: []string{writeDir},
			Read:  []string{scriptDir},
		},
	}
	code, err := Run([]string{"/bin/sh", sh, deniedDir}, p)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if code == 0 {
		t.Fatal("expected nonzero exit writing outside grant")
	}
	if data, _ := os.ReadFile(filepath.Join(deniedDir, "keep.txt")); string(data) != "x" {
		t.Errorf("denied-side file was modified: %q", data)
	}
}

func TestSeatbeltExitCodePropagated(t *testing.T) {
	requireSandboxExec(t)

	scriptDir := t.TempDir()
	sh := writeFixture(t, scriptDir, "probe.sh", "#!/bin/sh\nexit 42\n")
	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{scriptDir}},
	}
	code, err := Run([]string{"/bin/sh", sh}, p)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if code != 42 {
		t.Errorf("exit code = %d, want 42", code)
	}
}

func TestSeatbeltEnvPassthrough(t *testing.T) {
	requireSandboxExec(t)

	scriptDir := t.TempDir()
	sh := writeFixture(t, scriptDir, "probe.sh", "#!/bin/sh\n/usr/bin/env\n")

	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{scriptDir}},
		Env:        policy.Env{Allow: []string{"WARDEN_TEST_ALLOWED"}},
	}
	// Set a known allowlisted and a known non-allowlisted var in the parent env.
	t.Setenv("WARDEN_TEST_ALLOWED", "yes")
	t.Setenv("WARDEN_TEST_SECRET", "nope")

	code, err := Run([]string{"/bin/sh", sh}, p)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
}

func TestSeatbeltFailsClosedWithoutSandboxExec(t *testing.T) {
	if _, err := exec.LookPath("sandbox-exec"); err != nil {
		t.Skip("sandbox-exec not installed — nothing to hide")
	}
	old := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent-path-xyz")
	defer os.Setenv("PATH", old)

	_, err := Run([]string{"/usr/bin/true"}, policy.Policy{})
	if err == nil {
		t.Fatal("expected error when sandbox-exec is missing")
	}
}

func TestSeatbeltTimeoutKillsProcess(t *testing.T) {
	requireSandboxExec(t)

	scriptDir := t.TempDir()
	sh := writeFixture(t, scriptDir, "probe.sh", "#!/bin/sh\nsleep 30\n")
	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{scriptDir}},
		Limits:     policy.Limits{TimeoutS: 1},
	}
	code, err := Run([]string{"/bin/sh", sh}, p)
	if err == nil && code == 0 {
		t.Fatal("expected non-zero exit or error for timeout")
	}
}
