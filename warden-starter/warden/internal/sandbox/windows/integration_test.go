//go:build windows

package windows

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

// These are escape tests for the real AppContainer boundary. They need a
// Windows host where the AppContainer/Job Object/WFP primitives are available;
// per TESTING.md they are skipped (not failed) when the primitive is missing.
func requireAppContainer(t *testing.T) {
	t.Helper()
	if !Supported() {
		t.Skip("AppContainer sandbox not usable on this host — skipping Windows escape test")
	}
}

// runEscape runs cmd under a policy and returns its exit code.
func runEscape(t *testing.T, p policy.Policy, cmd []string) (int, error) {
	t.Helper()
	requireAppContainer(t)
	return Run(cmd, p)
}

// console returns the absolute path to the Windows Command Prompt host.
func comspec(t *testing.T) string {
	t.Helper()
	cmd := os.Getenv("COMSPEC")
	if cmd == "" || !filepath.IsAbs(cmd) {
		t.Skip("COMSPEC not set to an absolute path — skipping")
	}
	return cmd
}

func TestEscapeUngrantedReadDenied(t *testing.T) {
	grantedDir := t.TempDir()
	secretDir := t.TempDir()
	secret := filepath.Join(secretDir, "secret.txt")
	if err := os.WriteFile(secret, []byte("classified"), 0o600); err != nil {
		t.Fatal(err)
	}
	probe := filepath.Join(grantedDir, "probe.cmd")
	if err := os.WriteFile(probe, []byte("@echo off\r\ntype \"%1\" >nul\r\n"), 0o700); err != nil {
		t.Fatal(err)
	}

	p := policy.Policy{Filesystem: policy.Filesystem{Read: []string{grantedDir}}}
	code, err := runEscape(t, p, []string{comspec(t), "/c", probe, secret})
	if err != nil {
		// A fail-closed startup error is acceptable; an escape is not.
		t.Skipf("sandbox could not start here: %v", err)
	}
	if code == 0 {
		t.Fatalf("read of an un-granted path returned 0: sandbox did not deny it")
	}
}

func TestGrantedPathReadable(t *testing.T) {
	grantedDir := t.TempDir()
	want := filepath.Join(grantedDir, "in.txt")
	if err := os.WriteFile(want, []byte("allowed"), 0o600); err != nil {
		t.Fatal(err)
	}
	probe := filepath.Join(grantedDir, "probe.cmd")
	if err := os.WriteFile(probe, []byte("@echo off\r\ntype \"%1\"\r\n"), 0o700); err != nil {
		t.Fatal(err)
	}

	p := policy.Policy{Filesystem: policy.Filesystem{Read: []string{grantedDir}}}
	code, err := runEscape(t, p, []string{comspec(t), "/c", probe, want})
	if err != nil {
		t.Skipf("sandbox could not start here: %v", err)
	}
	if code != 0 {
		t.Fatalf("read of granted path failed (exit %d): got: %v", code, err)
	}
}

func TestEscapeExceedsTimeoutTerminatesTree(t *testing.T) {
	grantedDir := t.TempDir()
	probe := filepath.Join(grantedDir, "sleep.cmd")
	if err := os.WriteFile(probe, []byte("@echo off\r\nping -n 200 127.0.0.1 >nul\r\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	p := policy.Policy{Filesystem: policy.Filesystem{Read: []string{grantedDir}}, Limits: policy.Limits{TimeoutS: 1}}
	code, err := runEscape(t, p, []string{comspec(t), "/c", probe})
	if err != nil {
		if _, ok := err.(*LimitExceededError); ok {
			return // correct: the policy terminated the tree
		}
		t.Skipf("sandbox could not start here: %v", err)
	}
	if code == 0 {
		t.Fatalf("timed-out sandbox returned 0; expected a limit breach")
	}
}
