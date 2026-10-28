//go:build linux

package agentworkspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	linuxbackend "github.com/warden-sandbox/warden/internal/sandbox/linux"
)

func TestLinuxProjectionBoundary(t *testing.T) {
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		t.Skip("bubblewrap unavailable")
	}
	probe := exec.Command(bwrap, "--unshare-user", "--unshare-net", "--ro-bind", "/usr", "/usr", "--ro-bind", "/lib", "/lib", "--ro-bind", "/lib64", "/lib64", "/usr/bin/true")
	if out, err := probe.CombinedOutput(); err != nil {
		if os.Getenv("WARDEN_REQUIRE_AGENT_E2E") == "1" {
			t.Fatalf("required sandbox unavailable: %s", out)
		}
		t.Skipf("user/network namespace unavailable: %s", out)
	}
	s, path := fixture(t, "dotenv", "KEY=hidden-value\nDEBUG=false\n", "KEY")
	script := `set -eu
echo TARGET_STARTED
grep -q __WARDEN_REDACTED_ "$1/config"
if cat "$2" 2>/dev/null; then exit 21; fi
if printf hacked > "$1/.warden/NOTICE.md" 2>/dev/null; then exit 22; fi
sed -i 's/DEBUG=false/DEBUG=true/' "$1/config"
cat "$1/config"
`
	command := []string{"/usr/bin/sh", "-c", script, "sh", s.Dir, path}
	args, err := linuxbackend.BuildBwrapArgs(command, s.Policy)
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(bwrap, append(args, command...)...)
	out, err := child.CombinedOutput()
	if err != nil {
		t.Fatalf("projection child failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "TARGET_STARTED") || strings.Contains(string(out), "hidden-value") {
		t.Fatal("startup/secret exclusion assertion failed")
	}
	changes, err := s.Changes()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Apply(changes); err != nil {
		t.Fatal(err)
	}
	host, _ := os.ReadFile(filepath.Join(s.original, "config"))
	if !strings.Contains(string(host), "KEY=hidden-value") || !strings.Contains(string(host), "DEBUG=true") {
		t.Fatal("merge failed")
	}
}
