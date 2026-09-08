package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildWarden builds warden binary to a temp file and returns its path.
func buildWarden(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	name := "warden"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	out := filepath.Join(tmp, name)
	cmd := exec.Command("go", "build", "-o", out, ".")
	cmd.Dir = "."
	if outB, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build warden: %v\n%s", err, outB)
	}
	return out
}

func runWarden(t *testing.T, bin string, env []string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	// Inherit current env plus overrides
	cmd.Env = os.Environ()
	// Apply overrides
	for _, kv := range env {
		parts := strings.SplitN(kv, "=", 2)
		k := parts[0]
		// replace or append
		found := false
		for i, e := range cmd.Env {
			if strings.HasPrefix(e, k+"=") {
				cmd.Env[i] = kv
				found = true
				break
			}
		}
		if !found {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	// Force non-interactive for deterministic tests unless explicitly overridden
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("runWarden %v: %v", args, err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

func TestVersionRemainsClean(t *testing.T) {
	bin := buildWarden(t)
	stdout, stderr, code := runWarden(t, bin, nil, "--version")
	if code != 0 {
		t.Fatalf("--version exit %d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if stderr != "" {
		t.Errorf("--version should not write to stderr, got %q", stderr)
	}
	if !strings.HasPrefix(strings.TrimSpace(stdout), "warden version") {
		t.Errorf("--version output = %q, want prefix 'warden version'", stdout)
	}
	if strings.Contains(stdout, "██") || strings.Contains(stderr, "██") {
		t.Error("--version should not contain banner")
	}
	if strings.Contains(stdout, "\x1b[") {
		t.Error("--version should not contain ANSI")
	}
}

func TestHelpRemainsClean(t *testing.T) {
	bin := buildWarden(t)
	_, stderr, code := runWarden(t, bin, []string{"NO_COLOR=1"}, "--help")
	if code != 0 {
		t.Fatalf("--help exit %d", code)
	}
	if strings.Contains(stderr, "██") {
		t.Error("--help should not contain large banner block art")
	}
	if !strings.Contains(stderr, "WARDEN") {
		t.Error("--help should contain WARDEN header")
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Error("--help should contain Usage:")
	}
	if !strings.Contains(stderr, "warden run") {
		t.Error("--help should document run")
	}
	if !strings.Contains(stderr, "fails closed") {
		t.Error("--help should mention fail-closed")
	}
	if strings.Contains(stderr, "\x1b[") {
		t.Error("--help with NO_COLOR should not contain ANSI")
	}
}

func TestRunHelpClean(t *testing.T) {
	bin := buildWarden(t)
	_, stderr, code := runWarden(t, bin, []string{"NO_COLOR=1"}, "run", "--help")
	if code != 0 {
		t.Fatalf("run --help exit %d stderr=%q", code, stderr)
	}
	if strings.Contains(stderr, "██") {
		t.Error("run --help should not contain banner")
	}
	if !strings.Contains(stderr, "warden run") {
		t.Error("run --help should contain run usage")
	}
}

func TestBareInvocationNonTTYNoBanner(t *testing.T) {
	bin := buildWarden(t)
	tmp := t.TempDir()
	_, stderr, _ := runWarden(t, bin, []string{"NO_COLOR=1", "XDG_STATE_HOME=" + tmp, "WARDEN_NO_FIRST_RUN=1"}, "")
	// bare invocation is `warden` with no args -> we passed empty string as arg? Need to pass no args, but runWarden passes args slice; if we pass "" it is an arg.
	// Instead invoke with no args via exec directly
	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "XDG_STATE_HOME="+tmp, "WARDEN_NO_FIRST_RUN=1")
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	_ = cmd.Run()
	stderr = errBuf.String()
	if strings.Contains(stderr, "██") {
		t.Errorf("bare non-TTY should not contain large banner, got %q", stderr)
	}
	if !strings.Contains(stderr, "WARDEN") {
		t.Errorf("bare should contain WARDEN header, got %q", stderr)
	}
}

func TestColorDisabledOutput(t *testing.T) {
	bin := buildWarden(t)
	_, stderr, _ := runWarden(t, bin, []string{"NO_COLOR=1", "XDG_STATE_HOME=" + t.TempDir()}, "doctor")
	if strings.Contains(stderr, "\x1b[") {
		t.Errorf("doctor with NO_COLOR should not contain ANSI, got %q", stderr)
	}
	// Also check --help
	_, stderr2, _ := runWarden(t, bin, []string{"NO_COLOR=1"}, "--help")
	if strings.Contains(stderr2, "\x1b[") {
		t.Errorf("--help with NO_COLOR should not contain ANSI, got %q", stderr2)
	}
}

func TestNarrowTerminalFallback(t *testing.T) {
	bin := buildWarden(t)
	tmp := t.TempDir()
	_, stderr, _ := runWarden(t, bin, []string{"NO_COLOR=1", "COLUMNS=40", "WARDEN_TERM_WIDTH=40", "XDG_STATE_HOME=" + tmp, "WARDEN_NO_FIRST_RUN=1"}, "doctor")
	if strings.Contains(stderr, "██     ██") {
		t.Error("doctor with narrow terminal should fallback to compact header, not large banner")
	}
	// Should still contain WARDEN DOCTOR
	if !strings.Contains(stderr, "WARDEN DOCTOR") {
		t.Errorf("doctor narrow should still contain WARDEN DOCTOR, got %q", stderr)
	}
}

func TestNonInteractiveProgressFallback(t *testing.T) {
	bin := buildWarden(t)
	// CI env should force non-interactive
	_, stderr, _ := runWarden(t, bin, []string{"CI=true", "NO_COLOR=1", "XDG_STATE_HOME=" + t.TempDir()}, "doctor")
	// doctor doesn't use progress spinner, but ensure no spinner artifacts
	if strings.Contains(stderr, "⠋") || strings.Contains(stderr, "⠙") {
		t.Error("CI doctor should not contain spinner frames")
	}
}

func TestFailClosedMessaging(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("backend mismatch test is for non-windows hosts")
	}
	// Test that RefuseToRun error contains fail-closed messaging (via sandboxerr)
	// This is unit-level: run warden with auto backend forced to fail by using an isolated env
	// We can directly test the error string via the binary's doctor NOT READY case when backend unavailable.
	// Instead we test the sandboxerr directly via subprocess that triggers a RefuseToRun:
	// On this host bwrap is available, so auto will succeed. To force failure, use an explicit unknown backend
	// and check that error is not swallowed, and that doctor's NOT READY contains fails closed.
	bin := buildWarden(t)
	tmp := t.TempDir()

	// Use a command that is absolute on the current platform.
	cmd := "/usr/bin/true"
	if runtime.GOOS == "windows" {
		cmd = "C:\\Windows\\System32\\cmd.exe"
	}

	// Create a minimal policy that will trigger backend failure when we use --backend windows on linux
	policyPath := filepath.Join(tmp, "policy.yaml")
	_ = os.WriteFile(policyPath, []byte("command: [\""+cmd+"\"]\n"), 0o600)
	_, stderr, code := runWarden(t, bin, []string{"NO_COLOR=1"}, "run", "--policy", policyPath, "--backend", "windows", "--", cmd)
	if code == 0 {
		t.Fatalf("expected failure for windows backend on linux")
	}
	// Should contain either explicit OS mismatch or, if it were RefuseToRun, the fail-closed phrase.
	// At minimum ensure it mentions backend
	if !strings.Contains(stderr, "windows") && !strings.Contains(stderr, "backend") {
		t.Errorf("fail-closed error should mention backend, got %q", stderr)
	}
	// Also test that policy with missing file fails with clear message
	_, stderr2, code2 := runWarden(t, bin, []string{"NO_COLOR=1"}, "run", "--policy", "/nonexistent.yaml", "--", cmd)
	if code2 == 0 {
		t.Error("expected failure for missing policy")
	}
	if !strings.Contains(stderr2, "read policy") {
		t.Errorf("missing policy error should mention read policy, got %q", stderr2)
	}
}

func TestFirstRunWelcome(t *testing.T) {
	bin := buildWarden(t)
	tmp := t.TempDir()
	// Build a clean env that strips CI indicators so IsFirstRun doesn't
	// short-circuit before WARDEN_FORCE_FIRST_RUN takes effect.
	env := strippedEnv(os.Environ())
	env = append(env, "XDG_STATE_HOME="+tmp, "WARDEN_FORCE_FIRST_RUN=1", "NO_COLOR=1")
	// Force first-run via env, even though not TTY - invoke bare (no args)
	cmd := exec.Command(bin)
	cmd.Env = env
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	_ = cmd.Run()
	combined := outBuf.String() + errBuf.String()
	if !strings.Contains(combined, "Welcome to Warden") {
		t.Errorf("first-run should contain Welcome to Warden, got stdout=%q stderr=%q", outBuf.String(), errBuf.String())
	}
	if !strings.Contains(combined, "Policy enforcement") {
		t.Errorf("first-run should list capabilities, got %q", combined)
	}
	// Second run should not contain welcome (marker exists)
	cmd2 := exec.Command(bin)
	cmd2.Env = append(strippedEnv(os.Environ()), "XDG_STATE_HOME="+tmp, "NO_COLOR=1")
	var outBuf2, errBuf2 bytes.Buffer
	cmd2.Stdout = &outBuf2
	cmd2.Stderr = &errBuf2
	_ = cmd2.Run()
	combined2 := outBuf2.String() + errBuf2.String()
	if strings.Contains(combined2, "Welcome to Warden") {
		t.Errorf("second run should not contain welcome, got %q", combined2)
	}
}

// strippedEnv returns env with CI-indicator variables removed so
// IsFirstRun / IsCI don't short-circuit in subprocess tests.
func strippedEnv(env []string) []string {
	ciVars := map[string]bool{
		"CI": true, "GITHUB_ACTIONS": true, "GITLAB_CI": true,
		"JENKINS_URL": true, "TF_BUILD": true, "CIRCLECI": true,
		"BUILDKITE": true, "TEAMCITY_VERSION": true,
	}
	var out []string
	for _, e := range env {
		k := strings.SplitN(e, "=", 2)[0]
		if !ciVars[k] {
			out = append(out, e)
		}
	}
	return out
}

func TestFirstRunDoesNotInterfereWithRun(t *testing.T) {
	bin := buildWarden(t)
	tmp := t.TempDir()
	policyPath := filepath.Join(tmp, "policy.yaml")
	_ = os.WriteFile(policyPath, []byte("command: [\"/usr/bin/true\"]\n"), 0o600)
	// Set XDG to tmp with welcome marker already, then run a sandboxed command in CI mode (non-TTY)
	// It should not require interactive input and should succeed (or fail due to backend, but not due to first-run prompt)
	// We just ensure it doesn't hang or require input.
	_, stderr, code := runWarden(t, bin, []string{"XDG_STATE_HOME=" + tmp, "CI=true", "NO_COLOR=1", "WARDEN_NO_FIRST_RUN=1"}, "run", "--policy", policyPath, "--", "/usr/bin/true")
	// In this environment, /usr/bin/true under bwrap should succeed if bwrap available.
	// If it fails due to policy or backend, ensure it doesn't contain Welcome
	if strings.Contains(stderr, "Welcome to Warden") {
		t.Errorf("run should never show welcome, got %q", stderr)
	}
	// If bwrap not available, it might fail closed, but should contain fail-closed message if so
	if code != 0 && !strings.Contains(stderr, "warden") {
		t.Errorf("run failure should mention warden, got %q", stderr)
	}
}
