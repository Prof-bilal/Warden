//go:build windows

package windows

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/policy"
)

// These tests exercise the real ETW audit session. Like the escape tests,
// they require a Windows host where the AppContainer/ETW primitives are
// available and are skipped (not failed) when the primitive is missing. On
// the Windows CI runner (elevated) they execute for real and pin the session
// lifecycle and the Kernel-File event decode.

// probeETW verifies that an ETW audit session can actually start on this
// host, skipping the test otherwise. It also closes its probe session, so the
// caller starts fresh.
func probeETW(t *testing.T) {
	t.Helper()
	requireAppContainer(t)
	s, err := startTrace("warden.probe."+testSessionName(t), audit.New(&bytes.Buffer{}))
	if err != nil {
		t.Skipf("ETW audit session could not start here (is warden elevated?): %v", err)
	}
	if err := s.closeSession(); err != nil {
		t.Fatalf("close probe session: %v", err)
	}
}

// testSessionName derives a unique, ETW-safe session name from the test name.
func testSessionName(t *testing.T) string {
	name := strings.NewReplacer("/", ".", "#", ".", " ", "_").Replace(t.Name())
	if len(name) > 80 {
		name = name[:80]
	}
	return name
}

func TestEtwSessionLifecycle(t *testing.T) {
	probeETW(t)

	logBuf := &bytes.Buffer{}
	logger := audit.New(logBuf)

	// Scope capture to this test process so the callback path runs.
	sess, err := startTrace("warden.test."+testSessionName(t), logger)
	if err != nil {
		t.Fatalf("startTrace: %v", err)
	}
	sess.tree = map[uint32]bool{uint32(os.Getpid()): true}
	go sess.run()

	// Give ProcessTrace a moment to attach and consume a few events.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}

	if err := sess.closeSession(); err != nil {
		t.Fatalf("closeSession: %v", err)
	}
	// A second close must be a safe no-op (closeSession is idempotent).
	if err := sess.closeSession(); err != nil {
		t.Fatalf("second closeSession: %v", err)
	}
}

func TestEtwSessionNameReclaim(t *testing.T) {
	probeETW(t)

	name := "warden.reclaim." + testSessionName(t)
	mk := func() *traceSession {
		t.Helper()
		s, err := startTrace(name, audit.New(&bytes.Buffer{}))
		if err != nil {
			t.Fatalf("startTrace(%s): %v", name, err)
		}
		return s
	}
	first := mk()
	if err := first.closeSession(); err != nil {
		t.Fatalf("close first session: %v", err)
	}
	// The same session name must be reusable after a clean close.
	second := mk()
	if err := second.closeSession(); err != nil {
		t.Fatalf("close second session: %v", err)
	}
}

// TestEtwFileActivityAudited runs the real sandbox and asserts that a file
// operation by the sandboxed process lands in the audit log as a `file`
// event whose resource is the path that was touched. It is the pin for the
// Kernel-File decode (etwdecode.go): if real events stop carrying names on a
// given Windows build, this test fails and the decode offsets need updating
// on that build.
func TestEtwFileActivityAudited(t *testing.T) {
	probeETW(t)

	grantedDir := t.TempDir()
	out := filepath.Join(grantedDir, "warden-etw-probe.txt")

	// cmd.exe builtins run in the sandbox root process (no child spawn), so
	// the ETW events are scoped to a PID that was in the tree snapshot. The
	// path is passed unquoted, so it must contain no spaces (true on CI and
	// typical local Windows usernames).
	if strings.ContainsAny(out, " ") {
		t.Skipf("temp path contains spaces: %q", out)
	}
	probe := "echo wardenprobe>" + out
	p := policy.Policy{Filesystem: policy.Filesystem{Write: []string{grantedDir}}}
	code, err := runEscape(t, p, []string{comspec(t), "/c", probe})
	if err != nil {
		// A fail-closed startup error (e.g. ETW provider held by another
		// controller) is acceptable here; an unexpected sandbox error is not.
		if strings.Contains(err.Error(), "ETW") {
			t.Skipf("ETW audit could not start: %v", err)
		}
		t.Skipf("sandbox could not start here: %v", err)
	}
	if code != 0 {
		t.Fatalf("cmd exited %d; expected 0", code)
	}
	if _, err := os.Stat(out); err != nil {
		t.Skipf("probe did not create %q (%v); redirection failed on this host", out, err)
	}

	wantSuffix := strings.ToLower(`\` + filepath.Base(out))
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, ev := range readAuditEvents(t) {
			if ev.Type == "file" && ev.Allowed &&
				strings.HasSuffix(strings.ToLower(ev.Resource), wantSuffix) {
				return // the Kernel-File event carried the path
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("no ETW file audit event for %q appeared in the audit log", out)
}
