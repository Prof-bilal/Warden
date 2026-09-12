//go:build linux

package linux

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/envfilter"
	"github.com/warden-sandbox/warden/internal/policy"
)

// The integration tests below are escape tests for the real sandbox
// boundary. They need bwrap + unprivileged user namespaces and are skipped
// (not failed) on machines without them, per TESTING.md.
//
// False-positive hardening (plan §5): every denial test must first prove the
// target actually STARTED inside the sandbox (positive control) before
// asserting a security result. runSandbox/runSandboxWithEnv gate on
// requireTargetStarted — the same startupMarker pattern the darwin tests use —
// so "target never launched" can never be reported as a sandbox PASS.

// startupMarker is printed by a probe target; seeing it on stdout proves the
// sandbox launched the child and the child executed.
const startupMarker = "WARDEN_SANDBOX_UP"

// runBwrapRaw is the low-level bwrap chain without the positive-control gate:
// requireBwrap + BuildBwrapArgs + exec. Only the positive control and the
// gated runners below may use it.
func runBwrapRaw(t *testing.T, p policy.Policy, cmd []string) (string, int, error) {
	t.Helper()
	bwrap := requireBwrap(t)
	args, err := BuildBwrapArgs(cmd, p)
	if err != nil {
		t.Fatal(err)
	}
	sub := exec.Command(bwrap, append(args, cmd...)...)
	out, err := sub.CombinedOutput()
	if err == nil {
		return string(out), 0, nil
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return string(out), 0, err
	}
	return string(out), exitErr.ExitCode(), nil
}

// requireTargetStarted is the positive control (plan §5): it runs a
// marker-printing script under the production bwrap chain and FAILS (never
// skips) when the target does not actually start. A security denial asserted
// without this control could be a dead sandbox passing for the wrong reason —
// the exact false-positive mode the darwin tests already guard against.
func requireTargetStarted(t *testing.T) {
	t.Helper()
	requireBwrap(t)
	dir := t.TempDir()
	script := writeFixture(t, dir, "startup-probe.sh", "#!/bin/sh\necho "+startupMarker+"\n")
	p := policy.Policy{Filesystem: policy.Filesystem{Read: []string{dir}}}
	out, code, err := runBwrapRaw(t, p, []string{"/usr/bin/sh", script})
	if err != nil {
		t.Fatalf("positive control: sandboxed target failed to start: %v\noutput:\n%s", err, out)
	}
	if code != 0 {
		t.Fatalf("positive control: startup probe exited %d, want 0\noutput:\n%s", code, out)
	}
	if !strings.Contains(out, startupMarker) {
		t.Fatalf("positive control: startup marker missing — target never really ran\noutput:\n%s", out)
	}
}

func requireBwrap(t *testing.T) string {
	t.Helper()
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		t.Skip("bwrap not installed — skipping sandbox integration test")
	}
	if err := exec.Command(bwrap, "--unshare-user", "--unshare-net", "--uid", "0", "--gid", "0", "--ro-bind", "/usr", "/usr", "--ro-bind", "/lib", "/lib", "--ro-bind", "/lib64", "/lib64", "/usr/bin/true").Run(); err != nil {
		t.Skipf("bwrap sandbox unusable on this host (%v) — skipping", err)
	}
	return bwrap
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

// runSandbox runs cmd inside a sandbox built from p and returns output and
// exit code. It first proves the target can start (positive control, plan §5)
// so a dead sandbox can never be reported as a security PASS.
func runSandbox(t *testing.T, p policy.Policy, cmd []string) (string, int, error) {
	t.Helper()
	requireTargetStarted(t)
	return runBwrapRaw(t, p, cmd)
}

// runSandboxWithEnv is like runSandbox but lets the test control the parent
// environment so env-passthrough behavior can be pinned down. Like runSandbox,
// it gates on the positive control first.
func runSandboxWithEnv(t *testing.T, p policy.Policy, cmd, parentEnv []string) (string, int, error) {
	t.Helper()
	requireTargetStarted(t)
	bwrap := requireBwrap(t)
	args, err := BuildBwrapArgs(cmd, p)
	if err != nil {
		t.Fatal(err)
	}
	sub := exec.Command(bwrap, append(args, cmd...)...)
	sub.Env = envfilter.Filter(parentEnv, p.EnvAllowlist())
	out, err := sub.CombinedOutput()
	if err == nil {
		return string(out), 0, nil
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return string(out), 0, err
	}
	return string(out), exitErr.ExitCode(), nil
}

// TestSandboxPositiveControlStartup is the explicit, named positive control:
// the sandbox must launch a marker-printing target and surface its output.
// If this fails, every other result from this package is uninterpretable.
func TestSandboxPositiveControlStartup(t *testing.T) {
	requireTargetStarted(t)
}

func TestSandboxReadGrantAccessible(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "secret.txt"), []byte("allowed"), 0o644); err != nil {
		t.Fatal(err)
	}

	scriptDir := t.TempDir()
	sh := writeFixture(t, scriptDir, "probe.sh", "#!/bin/sh\ncat \"$1\"\n")
	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{dataDir, scriptDir}},
	}
	cmd := []string{"/usr/bin/sh", sh, filepath.Join(dataDir, "secret.txt")}

	out, code, err := runSandbox(t, p, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit %d, output: %s", code, out)
	}
	if !strings.Contains(out, "allowed") {
		t.Errorf("expected to read granted path, got: %q", out)
	}
}

func TestSandboxUnlistedPathInvisible(t *testing.T) {
	// A sibling directory that is not granted must be invisible inside the
	// sandbox — not merely permission-denied.
	granted := t.TempDir()
	denied := t.TempDir()
	os.WriteFile(filepath.Join(denied, "topsecret.txt"), []byte("nope"), 0o644)

	scriptDir := t.TempDir()
	sh := writeFixture(t, scriptDir, "probe.sh", "#!/bin/sh\nls \"$1\" 2>/dev/null || echo NOT_VISIBLE\n")
	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{granted, scriptDir}},
	}
	cmd := []string{"/usr/bin/sh", sh, denied}

	out, code, err := runSandbox(t, p, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit %d, output: %s", code, out)
	}
	if !strings.Contains(out, "NOT_VISIBLE") {
		t.Errorf("unlisted path should be invisible, got: %q", out)
	}
}

func TestSandboxWriteGrantWritable(t *testing.T) {
	writeDir := t.TempDir() // created outside sandbox; mount source must exist
	scriptDir := t.TempDir()
	sh := writeFixture(t, scriptDir, "probe.sh", "#!/bin/sh\necho made > \"$1/f.txt\" && cat \"$1/f.txt\"\n")
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Write: []string{writeDir},
			Read:  []string{scriptDir},
		},
	}
	cmd := []string{"/usr/bin/sh", sh, writeDir}

	out, code, err := runSandbox(t, p, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit %d, output: %s", code, out)
	}
	if !strings.Contains(out, "made") {
		t.Errorf("expected write within grant to work, got: %q", out)
	}
	// The file must actually exist on the host (the mount is the same dir).
	if _, err := os.Stat(filepath.Join(writeDir, "f.txt")); err != nil {
		t.Errorf("file not created in write grant: %v", err)
	}
}

func TestSandboxWriteOutsideGrantDenied(t *testing.T) {
	writeDir := t.TempDir()
	deniedDir := t.TempDir()
	os.WriteFile(filepath.Join(deniedDir, "keep.txt"), []byte("x"), 0o644)

	scriptDir := t.TempDir()
	sh := writeFixture(t, scriptDir, "probe.sh", "#!/bin/sh\necho evil > \"$1/keep.txt\"\n")
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Write: []string{writeDir},
			Read:  []string{scriptDir},
		},
	}
	cmd := []string{"/usr/bin/sh", sh, deniedDir}

	_, code, err := runSandbox(t, p, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if code == 0 {
		t.Fatalf("expected nonzero exit writing outside grant")
	}
	if data, _ := os.ReadFile(filepath.Join(deniedDir, "keep.txt")); string(data) != "x" {
		t.Errorf("granted-side file was modified: %q", data)
	}
}

func TestSandboxExitCodePropagated(t *testing.T) {
	p := policy.Policy{Filesystem: policy.Filesystem{Read: []string{t.TempDir()}}}
	// Grant a dir we don't need so the policy is non-trivially exercised.
	_, code, err := runSandbox(t, p, []string{"/usr/bin/sh", "-c", "exit 42"})
	if err != nil {
		t.Fatal(err)
	}
	if code != 42 {
		t.Errorf("exit code = %d, want 42", code)
	}
}

func TestSandboxEnvPassthrough(t *testing.T) {
	scriptDir := t.TempDir()
	// Print every env var that exists; unset ones simply don't appear.
	sh := writeFixture(t, scriptDir, "probe.sh", "#!/bin/sh\n/usr/bin/env\n")

	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{scriptDir}},
		Env:        policy.Env{Allow: []string{"WARDEN_ALLOWED"}},
	}
	cmd := []string{"/usr/bin/sh", sh}

	parentEnv := []string{
		"WARDEN_ALLOWED=yes",
		"WARDEN_SECRET=nope",
	}

	out, code, err := runSandboxWithEnv(t, p, cmd, parentEnv)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit %d, output: %s", code, out)
	}
	if !strings.Contains(out, "WARDEN_ALLOWED=yes") {
		t.Errorf("allowlisted env var missing from sandbox env:\n%s", out)
	}
	if strings.Contains(out, "WARDEN_SECRET") {
		t.Errorf("non-allowlisted env var leaked into sandbox:\n%s", out)
	}
}

func TestRunFailsLoudWithoutBwrap(t *testing.T) {
	// bwrap is usually present on this machine; to exercise the
	// missing-bwrap path we hide it from PATH. The failure must be a clear
	// error, never a silent unsandboxed run.
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("bwrap not installed — nothing to hide")
	}
	old := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent-path-xyz")
	defer os.Setenv("PATH", old)

	_, err := Run([]string{"/usr/bin/true"}, policy.Policy{})
	if err == nil {
		t.Fatal("expected error when bwrap is missing")
	}
	if !strings.Contains(err.Error(), "sandbox backend unavailable") {
		t.Errorf("error should report sandbox backend unavailable: %v", err)
	}
	if !strings.Contains(err.Error(), "fails closed by design") {
		t.Errorf("error should explain fails closed by design: %v", err)
	}
}
