package main

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestDoctorProve(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux containment proof")
	}
	requireLinuxContainment(t)
	out, stderr, code := runWarden(t, buildWarden(t), []string{"XDG_STATE_HOME=" + t.TempDir()}, "doctor", "--prove")
	if code != 0 {
		t.Fatalf("doctor proof exited %d\n%s\n%s", code, out, stderr)
	}
	for _, label := range []string{"secret-file read", "write outside grant", "blocked network host", "environment sentinel"} {
		if !strings.Contains(out, "PASS "+label+"\n") {
			t.Errorf("missing pass for %s: %q", label, out)
		}
	}
}

// On non-Linux Go targets the native CI hosts provide their own containment.
func requireLinuxContainment(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		return
	}
	probe := exec.Command("bwrap", "--ro-bind", "/", "/", "--unshare-net", "--", "/usr/bin/true")
	if output, err := probe.CombinedOutput(); err != nil {
		if os.Getenv("WARDEN_REQUIRE_AGENT_E2E") == "1" {
			t.Fatalf("cannot run mandatory Linux containment proof: %v: %s", err, output)
		}
		t.Skipf("bubblewrap unavailable: %v", err)
	}
}
