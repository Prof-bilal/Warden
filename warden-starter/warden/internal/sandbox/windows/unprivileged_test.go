//go:build windows

package windows

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"

	"github.com/warden-sandbox/warden/internal/policy"
)

// TestUnprivilegedRefusesNetworkAllowlist: without WFP the proxy-only egress
// rule cannot be enforced, so a policy with network.allow must fail closed.
func TestUnprivilegedRefusesNetworkAllowlist(t *testing.T) {
	p := policy.Policy{Network: policy.Network{Allow: []string{"api.github.com"}}}
	err := checkUnprivilegedPolicy(p)
	if err == nil || !strings.Contains(err.Error(), "network.allow") {
		t.Fatalf("checkUnprivilegedPolicy = %v, want a network.allow refusal", err)
	}
	if err := checkUnprivilegedPolicy(policy.Policy{}); err != nil {
		t.Fatalf("policy without network grants refused: %v", err)
	}
}

// TestEscapeUnprivilegedNetworkBlocked: in unprivileged mode the AppContainer
// has no network capability, so direct egress, including to a loopback
// listener on the host, must fail.
func TestEscapeUnprivilegedNetworkBlocked(t *testing.T) {
	if wfpSupported() == nil || !unprivilegedOptIn() {
		t.Skip("only meaningful in unprivileged mode (" + unprivilegedEnv + "=1, no WFP)")
	}
	curl := filepath.Join(os.Getenv("SystemRoot"), "System32", "curl.exe")
	if _, err := os.Stat(curl); err != nil {
		t.Skip("curl.exe not available")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	accepted := make(chan struct{}, 1)
	go func() {
		if c, err := ln.Accept(); err == nil {
			c.Close()
			accepted <- struct{}{}
		}
	}()

	// Positive control: the target itself starts in the sandbox.
	if code, err := runEscape(t, policy.Policy{}, []string{curl, "--version"}); err != nil || code != 0 {
		t.Fatalf("positive control: curl --version in sandbox = %d, %v", code, err)
	}
	for _, url := range []string{"http://" + ln.Addr().String(), "http://1.1.1.1"} {
		code, err := Run([]string{curl, "--noproxy", "*", "-sS", "-m", "5", "-o", "NUL", url}, policy.Policy{})
		if err != nil {
			t.Fatalf("run curl %s: %v", url, err)
		}
		if code == 0 {
			t.Fatalf("sandboxed curl reached %s", url)
		}
	}
	select {
	case <-accepted:
		t.Fatal("sandboxed process connected to a host loopback listener")
	case <-time.After(200 * time.Millisecond):
	}
}

// TestGrantsRestoredAfterRun: the per-run capability ACEs must be removed and
// the original DACL (including inheritance) put back once the target exits.
func TestGrantsRestoredAfterRun(t *testing.T) {
	dir, err := filepath.Abs(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	before, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	parent := filepath.Dir(dir)
	parentBefore, err := windows.GetNamedSecurityInfo(parent, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	p := policy.Policy{Filesystem: policy.Filesystem{Write: []string{dir}}}
	if code, err := runEscape(t, p, []string{comspec(t), "/c", "echo x> " + filepath.Join(dir, "f.txt")}); err != nil || code != 0 {
		t.Fatalf("sandboxed write = %d, %v", code, err)
	}
	after, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	if before.String() != after.String() {
		t.Fatalf("DACL not restored:\nbefore %s\nafter  %s", before, after)
	}
	if strings.Contains(after.String(), "S-1-15-3-1024-") {
		t.Fatalf("capability ACE left behind: %s", after)
	}
	// The ancestor read-attributes grant (ancestors.go) must be undone too,
	// including the DACL's auto-inherited flag.
	parentAfter, err := windows.GetNamedSecurityInfo(parent, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	if parentBefore.String() != parentAfter.String() {
		t.Fatalf("ancestor DACL not restored:\nbefore %s\nafter  %s", parentBefore, parentAfter)
	}
}
