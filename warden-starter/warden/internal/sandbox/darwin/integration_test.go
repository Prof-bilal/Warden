//go:build darwin

package darwin

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/proxy"
)

// TestMain turns this test binary into a real __proxy-bridge when it is
// re-exec'd with that argument. darwin.Run uses os.Executable() as the
// bridge executable, so under `go test` the child must handle the bridge
// protocol in-process — otherwise the nested binary is just the test binary
// run with unknown flags.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "__proxy-bridge" {
		socket, listen, target, err := proxy.ParseBridgeArgs(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "test bridge: %v\n", err)
			os.Exit(2)
		}
		code, err := proxy.RunBridge(socket, listen, target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "test bridge: %v\n", err)
			os.Exit(1)
		}
		os.Exit(code)
	}
	os.Exit(m.Run())
}

// requireSandboxExec proves sandbox-exec can actually run a target process
// through the production profile and bridge chain. It FAILS (never skips)
// when the host cannot run sandboxed targets — a skip here is what made
// previous CI green while every real assertion was silently untested.
//
// The failure path runs a diagnostic ladder so a red run explains itself:
//
//	A. a minimal hand-written profile executing /bin/true directly (isolates
//	   sandbox-exec/host from the generated profile),
//	B. the production profile executing /bin/true directly, no bridge (so a
//	   failure here is the generated profile, not the bridge or env filter),
//	C. the full Run() path (generated profile + in-process bridge + sh),
//
// and on any failure dumps recent kernel sandbox denials from the unified
// log.
func requireSandboxExec(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sandbox-exec"); err != nil {
		t.Fatalf("sandbox-exec not installed: %v", err)
	}

	// A: minimal permissive profile, direct exec, no bridge. /usr/bin/true,
	// not /bin/true: macOS has no /bin/true (unlike Linux).
	minProfile := "(version 1)\n(deny default)\n" +
		"(allow process-exec)\n(allow process-fork)\n" +
		"(allow file-read* (subpath \"/\"))\n" +
		"(allow file-map-executable (subpath \"/\"))\n" +
		"(allow file-write* (subpath \"/dev/null\"))\n"
	minPath := filepath.Join(t.TempDir(), "minimal.sb")
	if err := os.WriteFile(minPath, []byte(minProfile), 0o644); err != nil {
		t.Fatalf("write minimal profile: %v", err)
	}
	if out, err := exec.Command("sandbox-exec", "-f", minPath, "/usr/bin/true").CombinedOutput(); err != nil {
		t.Fatalf("preflight A: sandbox-exec cannot exec /usr/bin/true under a minimal profile: %v\noutput:\n%s\n%s",
			err, out, sandboxDenialLog())
	}

	// B: production profile, direct exec, no bridge, no env filtering.
	// The socket path is a fixture string; no proxy is started for this step.
	tmp := t.TempDir()
	profile, err := BuildSeatbeltProfile([]string{"/usr/bin/true"}, policy.Policy{}, filepath.Join(tmp, "egress.sock"))
	if err != nil {
		t.Fatalf("preflight B: build profile: %v", err)
	}
	profilePath := filepath.Join(tmp, "production.sb")
	if err := os.WriteFile(profilePath, []byte(profile), 0o644); err != nil {
		t.Fatalf("write production profile: %v", err)
	}
	if out, err := exec.Command("sandbox-exec", "-f", profilePath, "/usr/bin/true").CombinedOutput(); err != nil {
		// B is a strict superset of A's grants, yet A runs and B aborts, so
		// some rule B *adds* offends sandbox-exec. Two probes name it:
		//
		// 1. Forward: A + B's network/mach-lookup rules. If this fails, the
		//    offender is in that block (the only non-filesystem rules B adds).
		// 2. Reverse: B minus one rule group per variant, plus a safety net
		//    (A's blanket root map + process-exec/fork) so no variant can
		//    fail for missing exec mechanics. A variant that RUNS means the
		//    dropped group contained the offending rule.
		safety := "(allow process-exec)\n(allow process-fork)\n(allow file-map-executable (subpath \"/\"))\n"
		findings := make([]string, 0, 20)

		networkBlock := ""
		for _, ln := range strings.Split(profile, "\n") {
			if strings.Contains(ln, "network-") || strings.Contains(ln, "mach-lookup") {
				networkBlock += ln + "\n"
			}
		}
		fpath := filepath.Join(tmp, "bisect-0-forward-a-plus-network.sb")
		if werr := os.WriteFile(fpath, []byte(minProfile+networkBlock), 0o644); werr != nil {
			t.Fatalf("write forward bisect profile: %v", werr)
		}
		fout, ferr := exec.Command("sandbox-exec", "-f", fpath, "/usr/bin/true").CombinedOutput()
		fstatus := "FAIL => offender is in the network/mach block"
		if ferr == nil {
			fstatus = "OK => offender is a filesystem rule"
		}
		findings = append(findings, fmt.Sprintf("  [%s] A + B network/mach rules => err=%v out=%q\nnetwork/mach block:\n%s", fstatus, ferr, strings.TrimSpace(string(fout)), networkBlock))

		groups := []struct {
			name    string
			markers []string
		}{
			{"process-exec/fork/signal/info", []string{"process-"}},
			{"sysctl-read", []string{"sysctl-read"}},
			{"mach-lookup", []string{"mach-lookup"}},
			{"file-read-metadata", []string{"file-read-metadata"}},
			{"ipc-posix-shm/sem", []string{"ipc-posix-"}},
			{"iokit-open", []string{"iokit-open"}},
			{"/dev basics", []string{"/dev/null", "/dev/zero", "/dev/urandom", "/dev/random", "/dev/ttys", "/dev/fd"}},
			{"/etc", []string{"subpath \"/etc\"", "subpath \"/private/etc\""}},
			{"all file-map-executable", []string{"file-map-executable"}},
			{"runtime file-read* subpaths", []string{"file-read* (subpath"}},
			{"scratch file-write*", []string{"file-write* (subpath \"/tmp\"", "file-write* (subpath \"/private/tmp\"", "file-write* (subpath \"/var/folders\"", "file-write* (subpath \"/private/var/folders\""}},
			{"network rules", []string{"network-"}},
			{"socket path grants", []string{"egress.sock"}},
			{"dtracehelper", []string{"dtracehelper"}},
		}
		drop := func(profile string, markers []string) string {
			lines := strings.Split(profile, "\n")
			kept := make([]string, 0, len(lines))
			for _, ln := range lines {
				matched := false
				for _, m := range markers {
					if strings.Contains(ln, m) {
						matched = true
						break
					}
				}
				if !matched {
					kept = append(kept, ln)
				}
			}
			return strings.Join(kept, "\n")
		}
		for i, g := range groups {
			vpath := filepath.Join(tmp, fmt.Sprintf("bisect-%d-minus-%s.sb", i+1, strings.ReplaceAll(strings.ReplaceAll(g.name, " ", "-"), "/", "_")))
			if werr := os.WriteFile(vpath, []byte(drop(profile, g.markers)+safety), 0o644); werr != nil {
				t.Fatalf("write reverse bisect profile: %v", werr)
			}
			vout, verr := exec.Command("sandbox-exec", "-f", vpath, "/usr/bin/true").CombinedOutput()
			status := "FAIL"
			if verr == nil {
				status = "OK <-- offending rule is in this group"
			}
			findings = append(findings, fmt.Sprintf("  [%s] B minus %q => err=%v out=%q", status, g.name, verr, strings.TrimSpace(string(vout))))
		}
		// Also dump the newest crash report body — dyld aborts record the
		// exact failing operation there.
		crash := "(none)"
		if c, cerr := exec.Command("/bin/sh", "-c", `newest=$(ls -t ~/Library/Logs/DiagnosticReports/ 2>/dev/null | head -1); echo "report: $newest"; head -c 4000 "$HOME/Library/Logs/DiagnosticReports/$newest" 2>/dev/null`).Output(); cerr == nil {
			crash = strings.TrimSpace(string(c))
		}
		t.Fatalf("preflight B: production profile cannot exec /usr/bin/true directly: %v\nprofile:\n%s\noutput:\n%s\nbisect findings:\n%s\nrecent crash reports: %s\n%s",
			err, profile, out, strings.Join(findings, "\n"), crash, sandboxDenialLog())
	}

	// C: the real chain — generated profile, in-process bridge,
	// marker-printing target. This is the failure mode that used to be
	// invisible (silent child death, exit -1).
	dir := t.TempDir()
	script := writeScript(t, dir, "probe.sh", "#!/bin/sh\necho "+startupMarker+"\n")
	code, out, err := runSandboxed(t, []string{"/bin/sh", script}, policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{dir}},
	})
	if err != nil {
		t.Fatalf("preflight C: full Run() chain failed: %v\noutput:\n%s\n%s", err, out, sandboxDenialLog())
	}
	if code != 0 {
		t.Fatalf("preflight C: target exited %d, want 0\noutput:\n%s\n%s", code, out, sandboxDenialLog())
	}
	if !strings.Contains(out, startupMarker) {
		t.Fatalf("preflight C: startup marker missing — target never really ran\noutput:\n%s\n%s", out, sandboxDenialLog())
	}
}

// sandboxDenialLog returns recent kernel sandbox denial messages, or a
// note that the log could not be read. It is diagnostics for test output,
// never an assertion.
func sandboxDenialLog() string {
	out, err := exec.Command("log", "show", "--last", "30s",
		"--predicate", `eventMessage CONTAINS "deny(1)"`,
		"--style", "compact").Output()
	if err != nil {
		return fmt.Sprintf("(unified log unavailable: %v)", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > 40 {
		lines = append(lines[:40], "...(truncated)")
	}
	if len(lines) == 1 && lines[0] == "" {
		return "(no sandbox denial messages in the last 30s)"
	}
	return "kernel sandbox log (last 30s):\n" + strings.Join(lines, "\n")
}

const startupMarker = "WARDEN_SANDBOX_UP"

// denyRoot returns a directory outside every always-granted path so
// Seatbelt denials can actually fire there. Tests must not place
// denial-assertion fixtures under t.TempDir(): /var/folders (and /tmp) are
// unconditionally granted as scratch space, so a denial could never fire
// and the test would pass for the wrong reason. The package cwd lives
// under the checkout (never granted by the profile).
func denyRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve cwd: %v", err)
	}
	dir := filepath.Join(cwd, "testdata", "deny", t.Name()+"-"+fmt.Sprint(os.Getpid()))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create deny fixture dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Dir(dir)) })
	return dir
}

// writeScript writes an executable script into dir and returns its path.
func writeScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// runSandboxed invokes Run with stdout/stderr captured so tests can assert
// on the target's output. It is bridge infrastructure, not part of the
// security assertions.
func runSandboxed(t *testing.T, cmd []string, p policy.Policy) (int, string, error) {
	t.Helper()
	devNull, err := os.OpenFile(os.DevNull, os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("open devnull: %v", err)
	}
	defer devNull.Close()

	// Run targets the real process stdio; swap them for the duration.
	oldIn, oldOut, oldErr := os.Stdin, os.Stdout, os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdin, os.Stdout, os.Stderr = devNull, w, w
	code, runErr := func() (int, error) {
		defer func() { os.Stdin, os.Stdout, os.Stderr = oldIn, oldOut, oldErr }()
		return Run(cmd, p)
	}()
	w.Close()
	data, _ := io.ReadAll(r)
	r.Close()
	return code, string(data), runErr
}

func TestSeatbeltBlocksUngrantedRead(t *testing.T) {
	requireSandboxExec(t)

	// The secret lives outside every granted path so the read denial can
	// actually fire (t.TempDir() under /var/folders is always granted).
	secretDir := denyRoot(t)
	if err := os.WriteFile(filepath.Join(secretDir, "secret.txt"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	scriptDir := t.TempDir()
	script := writeScript(t, scriptDir, "probe.sh", "#!/bin/sh\necho "+startupMarker+"\ncat \"$1\"\n")

	// Grant only the script dir — reading secretDir must fail.
	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{scriptDir}},
	}
	code, out, err := runSandboxed(t, []string{"/bin/sh", script, filepath.Join(secretDir, "secret.txt")}, p)
	if err != nil {
		t.Fatalf("Run: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, startupMarker) {
		t.Fatalf("target did not start (marker missing) — denial is not proven\noutput:\n%s", out)
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
	script := writeScript(t, scriptDir, "probe.sh",
		"#!/bin/sh\necho "+startupMarker+"\ncat \"$1\"\n")
	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{dataDir, scriptDir}},
	}
	code, out, err := runSandboxed(t, []string{"/bin/sh", script, filepath.Join(dataDir, "ok.txt")}, p)
	if err != nil {
		t.Fatalf("Run: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, startupMarker) {
		t.Fatalf("target did not start (marker missing)\noutput:\n%s", out)
	}
	if code != 0 {
		t.Fatalf("exit %d, want 0\noutput:\n%s", code, out)
	}
	if !strings.Contains(out, "allowed") {
		t.Fatalf("granted read did not return file contents\noutput:\n%s", out)
	}
}

func TestSeatbeltWriteGrantWritable(t *testing.T) {
	requireSandboxExec(t)

	writeDir := t.TempDir()
	scriptDir := t.TempDir()
	sh := writeScript(t, scriptDir, "probe.sh",
		"#!/bin/sh\necho "+startupMarker+"\necho made > \"$1/f.txt\" && cat \"$1/f.txt\"\n")
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Write: []string{writeDir},
			Read:  []string{scriptDir},
		},
	}
	code, out, err := runSandboxed(t, []string{"/bin/sh", sh, writeDir}, p)
	if err != nil {
		t.Fatalf("Run: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, startupMarker) {
		t.Fatalf("target did not start (marker missing)\noutput:\n%s", out)
	}
	if code != 0 {
		t.Fatalf("exit %d, want 0\noutput:\n%s", code, out)
	}
	if data, err := os.ReadFile(filepath.Join(writeDir, "f.txt")); err != nil || string(data) != "made\n" {
		t.Errorf("file not created in write grant: %v (%q)", err, string(data))
	}
}

func TestSeatbeltWriteOutsideGrantDenied(t *testing.T) {
	requireSandboxExec(t)

	writeDir := t.TempDir()
	// The denied dir sits outside every granted path so the write denial
	// can actually fire (t.TempDir() under /var/folders is always granted).
	deniedDir := denyRoot(t)
	if err := os.WriteFile(filepath.Join(deniedDir, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	scriptDir := t.TempDir()
	sh := writeScript(t, scriptDir, "probe.sh",
		"#!/bin/sh\necho "+startupMarker+"\necho evil > \"$1/keep.txt\"\n")
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Write: []string{writeDir},
			Read:  []string{scriptDir},
		},
	}
	code, out, err := runSandboxed(t, []string{"/bin/sh", sh, deniedDir}, p)
	if err != nil {
		t.Fatalf("Run: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, startupMarker) {
		t.Fatalf("target did not start (marker missing) — denial is not proven\noutput:\n%s", out)
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
	sh := writeScript(t, scriptDir, "probe.sh", "#!/bin/sh\necho "+startupMarker+"\nexit 42\n")
	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{scriptDir}},
	}
	code, out, err := runSandboxed(t, []string{"/bin/sh", sh}, p)
	if err != nil {
		t.Fatalf("Run: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, startupMarker) {
		t.Fatalf("target did not start (marker missing)\noutput:\n%s", out)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42\noutput:\n%s", code, out)
	}
}

func TestSeatbeltEnvPassthrough(t *testing.T) {
	requireSandboxExec(t)

	scriptDir := t.TempDir()
	sh := writeScript(t, scriptDir, "probe.sh",
		"#!/bin/sh\necho "+startupMarker+"\nprintf 'ALLOWED=%s\\nSECRET=%s\\n' \"$WARDEN_TEST_ALLOWED\" \"$WARDEN_TEST_SECRET\"\n")

	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{scriptDir}},
		Env:        policy.Env{Allow: []string{"WARDEN_TEST_ALLOWED"}},
	}
	// Set a known allowlisted and a known non-allowlisted var in the parent env.
	t.Setenv("WARDEN_TEST_ALLOWED", "yes")
	t.Setenv("WARDEN_TEST_SECRET", "nope")

	code, out, err := runSandboxed(t, []string{"/bin/sh", sh}, p)
	if err != nil {
		t.Fatalf("Run: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, startupMarker) {
		t.Fatalf("target did not start (marker missing)\noutput:\n%s", out)
	}
	if code != 0 {
		t.Fatalf("exit %d, want 0\noutput:\n%s", code, out)
	}
	if !strings.Contains(out, "ALLOWED=yes") {
		t.Errorf("allowed env var missing in sandbox\noutput:\n%s", out)
	}
	if strings.Contains(out, "SECRET=nope") {
		t.Errorf("forbidden env var leaked into sandbox\noutput:\n%s", out)
	}
}

func TestSeatbeltFailsClosedWithoutSandboxExec(t *testing.T) {
	if _, err := exec.LookPath("sandbox-exec"); err != nil {
		t.Skip("sandbox-exec not installed — nothing to prove")
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
	sh := writeScript(t, scriptDir, "probe.sh", "#!/bin/sh\necho "+startupMarker+"\nsleep 30\n")
	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{scriptDir}},
		Limits:     policy.Limits{TimeoutS: 1},
	}
	code, out, err := runSandboxed(t, []string{"/bin/sh", sh}, p)
	if !strings.Contains(out, startupMarker) {
		t.Fatalf("target did not start (marker missing) — timeout not proven\noutput:\n%s", out)
	}
	if err == nil && code == 0 {
		t.Fatal("expected non-zero exit or error for timeout")
	}
}

// TestSeatbeltNetworkAllowedViaProxy proves the Seatbelt profile permits
// loopback egress to the bridge only, and that the egress proxy enforces
// network.allow: a proxy request flows end-to-end through the bridge.
func TestSeatbeltNetworkAllowedViaProxy(t *testing.T) {
	requireSandboxExec(t)

	// Host-side fake upstream the egress proxy will dial.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "EGRESS-OK")
	}))
	defer srv.Close()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	scriptDir := t.TempDir()
	sh := writeScript(t, scriptDir, "probe.sh",
		"#!/bin/sh\ncommand -v curl >/dev/null 2>&1 || { echo NO_CURL; exit 9; }\necho "+startupMarker+"\nexec curl -sS --max-time 10 -x http://127.0.0.1:18080 "+u.Host+"\n")

	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{scriptDir}},
		Network:    policy.Network{Allow: []string{u.Hostname()}},
	}
	code, out, err := runSandboxed(t, []string{"/bin/sh", sh}, p)
	if err != nil {
		t.Fatalf("Run: %v\noutput:\n%s", err, out)
	}
	if strings.Contains(out, "NO_CURL") {
		t.Fatalf("curl not available in sandbox PATH — test cannot run\noutput:\n%s", out)
	}
	if !strings.Contains(out, startupMarker) {
		t.Fatalf("target did not start (marker missing)\noutput:\n%s", out)
	}
	if code != 0 {
		t.Fatalf("exit %d, want 0\noutput:\n%s", code, out)
	}
	if !strings.Contains(out, "EGRESS-OK") {
		t.Fatalf("allowed network request did not reach upstream via proxy\noutput:\n%s", out)
	}
}

// TestSeatbeltNetworkDeniedByPolicy proves a non-allowlisted destination is
// actually blocked by the egress proxy (proxy error), not by startup
// failure: the marker and the denial both appear.
func TestSeatbeltNetworkDeniedByPolicy(t *testing.T) {
	requireSandboxExec(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "SHOULD-NOT-LEAK")
	}))
	defer srv.Close()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	scriptDir := t.TempDir()
	// --fail: a proxy 403 must exit nonzero, otherwise a denied request
	// looks like success with an error page on stdout.
	sh := writeScript(t, scriptDir, "probe.sh",
		"#!/bin/sh\ncommand -v curl >/dev/null 2>&1 || { echo NO_CURL; exit 9; }\necho "+startupMarker+"\nexec curl -fsS --max-time 10 -x http://127.0.0.1:18080 "+u.Host+"\n")

	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{scriptDir}},
		Network:    policy.Network{Allow: []string{"allowed.example.invalid"}},
	}
	code, out, err := runSandboxed(t, []string{"/bin/sh", sh}, p)
	if err != nil {
		t.Fatalf("Run: %v\noutput:\n%s", err, out)
	}
	if strings.Contains(out, "NO_CURL") {
		t.Fatalf("curl not available in sandbox PATH — test cannot run\noutput:\n%s", out)
	}
	if !strings.Contains(out, startupMarker) {
		t.Fatalf("target did not start (marker missing) — denial is not proven\noutput:\n%s", out)
	}
	if strings.Contains(out, "SHOULD-NOT-LEAK") {
		t.Fatalf("non-allowlisted destination was reachable\noutput:\n%s", out)
	}
	if code == 0 {
		t.Fatalf("expected nonzero exit from denied proxy request\noutput:\n%s", out)
	}
}

// TestSeatbeltDirectIPIsNotDialable proves raw outbound sockets to arbitrary
// destinations are denied by Seatbelt itself (no proxy in the middle): the
// target starts, the connect fails. --noproxy neutralizes the HTTP_PROXY/
// ALL_PROXY variables Warden injects, so curl attempts a direct connection
// to the httptest server; the profile only allows localhost:18080.
func TestSeatbeltDirectIPIsNotDialable(t *testing.T) {
	requireSandboxExec(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "DIRECT-LEAK")
	}))
	defer srv.Close()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	scriptDir := t.TempDir()
	sh := writeScript(t, scriptDir, "probe.sh",
		"#!/bin/sh\ncommand -v curl >/dev/null 2>&1 || { echo NO_CURL; exit 9; }\necho "+startupMarker+"\nexec curl --noproxy '*' -sS --max-time 10 "+u.Host+"\n")

	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{scriptDir}},
		Network:    policy.Network{Allow: []string{u.Hostname()}},
	}
	code, out, err := runSandboxed(t, []string{"/bin/sh", sh}, p)
	if err != nil {
		t.Fatalf("Run: %v\noutput:\n%s", err, out)
	}
	if strings.Contains(out, "NO_CURL") {
		t.Fatalf("curl not available in sandbox PATH — test cannot run\noutput:\n%s", out)
	}
	if !strings.Contains(out, startupMarker) {
		t.Fatalf("target did not start (marker missing) — denial is not proven\noutput:\n%s", out)
	}
	if strings.Contains(out, "DIRECT-LEAK") {
		t.Fatalf("direct non-proxy egress escaped the sandbox\noutput:\n%s", out)
	}
	if code == 0 {
		t.Fatalf("direct connect outside the bridge unexpectedly succeeded\noutput:\n%s", out)
	}
}

// TestSeatbeltLoopbackBeyondBridgeIsDenied proves network-inbound is scoped
// to the bridge port: the sandboxed process may not itself listen on an
// arbitrary loopback port and accept connections from the host.
func TestSeatbeltLoopbackBeyondBridgeIsDenied(t *testing.T) {
	requireSandboxExec(t)

	// Free port for the probe to attempt to listen on.
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pick port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	scriptDir := t.TempDir()
	sh := writeScript(t, scriptDir, "probe.sh",
		"#!/bin/sh\ncommand -v nc >/dev/null 2>&1 || { echo NO_NC; exit 9; }\necho "+startupMarker+"\nnc -l "+fmt.Sprint(port)+" >/dev/null 2>&1 &\nNC_PID=$!\nsleep 2\nif kill -0 $NC_PID 2>/dev/null; then echo LISTEN_ALIVE; kill $NC_PID 2>/dev/null; exit 7; fi\necho LISTEN_DEAD\n")

	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{scriptDir}},
	}
	_, out, err := runSandboxed(t, []string{"/bin/sh", sh}, p)
	if err != nil {
		t.Fatalf("Run: %v\noutput:\n%s", err, out)
	}
	if strings.Contains(out, "NO_NC") {
		t.Fatalf("nc not available in sandbox PATH — test cannot run\noutput:\n%s", out)
	}
	if !strings.Contains(out, startupMarker) {
		t.Fatalf("target did not start (marker missing) — denial is not proven\noutput:\n%s", out)
	}
	if strings.Contains(out, "LISTEN_ALIVE") {
		t.Fatalf("sandboxed process could listen on a non-bridge loopback port\noutput:\n%s", out)
	}
}
