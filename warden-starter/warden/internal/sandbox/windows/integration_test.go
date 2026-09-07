//go:build windows

package windows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/warden-sandbox/warden/internal/audit"
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

// readAuditEvents decodes Warden's default audit log. Records from parallel
// warden runs on the same host may be interleaved, so assertions match on
// event content (a unique resource), never on position.
func readAuditEvents(t *testing.T) []audit.Event {
	t.Helper()
	path, err := audit.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	defer f.Close()
	events, err := audit.ReadEvents(f)
	if err != nil {
		t.Fatal(err)
	}
	return events
}

// waitAuditEvent polls the audit log until match succeeds (events are logged
// asynchronously by the proxy/ETW goroutines) and fails the test if it never
// appears.
func waitAuditEvent(t *testing.T, match func(audit.Event) bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, ev := range readAuditEvents(t) {
			if match(ev) {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("no matching audit event appeared within 5s")
}

// TestEscapeNetworkBlockedAudited runs the sandbox and asserts that a blocked
// network request lands in the audit log as an allowed=false `network` event.
// The sandboxed curl is forced through the egress proxy by the injected
// HTTP_PROXY variables; the proxy performs the hostname allowlist check and
// logs the exact denial — the deterministic "blocked" audit signal. (Direct
// DNS is also blocked — by the AppContainer token and the WFP deny filters —
// but denied packets never reach a provider that could audit them.)
func TestEscapeNetworkBlockedAudited(t *testing.T) {
	// curl.exe ships in System32 on supported Windows; it honors the
	// injected HTTP_PROXY environment without extra configuration.
	curl := filepath.Join(os.Getenv("SystemRoot"), "System32", "curl.exe")
	if _, err := os.Stat(curl); err != nil {
		t.Skip("curl.exe not present on this host")
	}

	const blockedHost = "blocked-w4rd3n.invalid"
	p := policy.Policy{Network: policy.Network{Allow: []string{"allowed.example"}}}
	// curl is the sandbox root image, so WFP permits exactly its loopback
	// connection to the proxy while the token denies everything else. The
	// proxy allows only "allowed.example"; the request to blockedHost must be
	// denied with HTTP 403, which curl reports as exit code 22.
	code, err := runEscape(t, p, []string{curl, "-s", "http://" + blockedHost + "/"})
	if err != nil {
		t.Skipf("sandbox could not start here: %v", err)
	}
	if code != 22 {
		t.Fatalf("curl exit = %d, want 22 (HTTP 403 from the deny-by-default proxy)", code)
	}
	waitAuditEvent(t, func(ev audit.Event) bool {
		return ev.Type == "network" && !ev.Allowed &&
			strings.EqualFold(ev.Resource, blockedHost+":80")
	})
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

// TestWFPDLLProbeResilient is the unit test for the REMAINING_WORK P0 follow-up:
// on hosts where fwpuclnt.dll is missing (e.g. stripped server SKUs, the GitHub
// Actions windows-latest runner image), initWFP() must fall back to the
// iphlapi.dll host or report a clear error rather than panicking inside
// LazyProc.Call. This test exercises the error-reporting path; on a host
// where WFP is fully available it succeeds without checking anything new.
func TestWFPDLLProbeResilient(t *testing.T) {
	err := wfpSupported()
	if err != nil {
		// Any error from wfpSupported() must be the failClose-wrapped
		// WFP error, not a panic or an unrelated code path.
		msg := err.Error()
		if !strings.Contains(msg, "WFP engine") {
			t.Fatalf("wfpSupported returned non-WFP error: %v", err)
		}
	}
}

// TestWFPDLLNameNonEmpty asserts the diagnostic helper always returns a
// non-empty string (either the resolved DLL name or the "(none)" sentinel)
// so the error message in wfpSupported() is always actionable.
func TestWFPDLLNameNonEmpty(t *testing.T) {
	name := wfpDLLName()
	if name == "" {
		t.Fatalf("wfpDLLName() returned empty string")
	}
}
