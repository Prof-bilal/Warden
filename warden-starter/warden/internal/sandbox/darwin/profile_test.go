// Profile generation is pure Go, so these tests run on every OS that
// shares POSIX path semantics (they do not need sandbox-exec; the
// darwin-tagged integration tests exercise the real backend). The Seatbelt
// profile requires POSIX-absolute paths by design, so the path-shaped test
// cases cannot run on Windows.

package darwin

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

// skipOnWindows reports why the path-shaped profile tests cannot run there:
// filepath.IsAbs("/usr/bin/true") is false on Windows, so the constructor's
// POSIX-absolute requirement (correct for Seatbelt) reads as an error.
func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Seatbelt profiles use POSIX-absolute paths; not meaningful on Windows")
	}
}

func TestBuildSeatbeltProfileDenyDefaultAndGrants(t *testing.T) {
	skipOnWindows(t)
	readDir := "/allowed/read"
	writeDir := "/allowed/write"
	socket := "/tmp/warden-proxy/egress.sock"
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Read:  []string{readDir},
			Write: []string{writeDir},
		},
	}
	profile, err := BuildSeatbeltProfile([]string{"/usr/bin/true"}, p, socket)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"(deny default)",
		`(allow file-read* (subpath "/allowed/read"))`,
		`(allow file-read* (subpath "/allowed/write"))`,
		`(allow file-write* (subpath "/allowed/write"))`,
		`(allow network-outbound (remote ip "localhost:18080"))`,
		`(allow file-read* (literal "/tmp/warden-proxy/egress.sock"))`,
		`(allow file-write* (literal "/tmp/warden-proxy/egress.sock"))`,
	} {
		if !strings.Contains(profile, want) {
			t.Fatalf("profile missing %q\n%s", want, profile)
		}
	}
	// Write grant must not appear as the only permission on a non-granted path.
	if strings.Contains(profile, `(allow file-write* (subpath "/allowed/read"))`) {
		t.Fatal("read grant incorrectly also writable")
	}
}

// TestBuildSeatbeltProfileNoIPLiterals guards the original macOS 26 bug:
// Seatbelt only accepts * or localhost as the host part of a network
// address, so no generated rule may contain an IP literal.
func TestBuildSeatbeltProfileNoIPLiterals(t *testing.T) {
	skipOnWindows(t)
	profile, err := BuildSeatbeltProfile([]string{"/usr/bin/true"}, policy.Policy{}, "/tmp/warden-proxy/egress.sock")
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{"127.0.0.1", "0.0.0.0", "::1"} {
		if strings.Contains(profile, banned) {
			t.Fatalf("profile contains unsupported IP literal %q\n%s", banned, profile)
		}
	}
}

// TestBuildSeatbeltProfileStartupPrimitives asserts the operations a
// deny-default profile needs before the target's main() runs: exec/fork,
// root path resolution, dyld image mapping, and /dev basics. Their absence
// made the sandbox abort the target before any policy denial could be
// observed (and made denial tests pass for the wrong reason).
func TestBuildSeatbeltProfileStartupPrimitives(t *testing.T) {
	skipOnWindows(t)
	profile, err := BuildSeatbeltProfile([]string{"/usr/bin/true"}, policy.Policy{}, "/tmp/warden-proxy/egress.sock")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"(allow process-exec)",
		"(allow process-fork)",
		"(allow file-read-metadata)",
		// dyld's CacheFinder probes the root directory itself while locating
		// the shared cache; denial aborts the process before main() on
		// macOS 26.
		`(allow file-read* file-test-existence (literal "/"))`,
		`(allow file-map-executable (subpath "/usr"))`,
		`(allow file-map-executable (subpath "/System"))`,
		`(allow file-map-executable (subpath "/Library/Apple/usr/lib"))`,
		`(allow file-map-executable (subpath "/System/Library/PrivateFrameworks"))`,
		`(allow file-read* (subpath "/etc"))`,
		`(allow file-read* (subpath "/private/var/db/timezone"))`,
		`(allow file-read* (subpath "/Library/Preferences/Logging"))`,
		`(allow file-read* file-write-data file-ioctl (literal "/dev/dtracehelper"))`,
		`(allow ipc-posix-shm-read*)`,
		`(allow ipc-posix-sem)`,
		`(allow iokit-open (iokit-registry-entry-class "RootDomainUserClient"))`,
		`(allow file-read* file-write-data (literal "/dev/null"))`,
		`(allow file-read* (literal "/dev/urandom"))`,
	} {
		if !strings.Contains(profile, want) {
			t.Fatalf("profile missing startup primitive %q\n%s", want, profile)
		}
	}
}

// TestBuildSeatbeltProfileMapsWriteGrantImages: a target binary inside a
// write grant (e.g. a built binary in the project dir) must be mappable.
func TestBuildSeatbeltProfileMapsWriteGrantImages(t *testing.T) {
	skipOnWindows(t)
	writeDir := "/allowed/write"
	profile, err := BuildSeatbeltProfile([]string{writeDir + "/server"}, policy.Policy{
		Filesystem: policy.Filesystem{Write: []string{writeDir}},
	}, "/tmp/warden-proxy/egress.sock")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(profile, `(allow file-map-executable (subpath "/allowed/write"))`) {
		t.Fatalf("write grant not mappable as executable image\n%s", profile)
	}
}

func TestBuildSeatbeltProfileRejectsConflicts(t *testing.T) {
	skipOnWindows(t)
	dir := "/tmp/conflict"
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Read:  []string{dir},
			Write: []string{dir},
		},
	}
	_, err := BuildSeatbeltProfile([]string{"/usr/bin/true"}, p, "/tmp/sock")
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestBuildSeatbeltProfileRejectsWriteIntoRuntimeBase(t *testing.T) {
	skipOnWindows(t)
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Write: []string{"/usr/local"},
		},
	}
	_, err := BuildSeatbeltProfile([]string{"/usr/bin/true"}, p, "/tmp/sock")
	if err == nil {
		t.Fatal("expected runtime-base write rejection")
	}
}

func TestBuildSeatbeltProfileRequiresAbsoluteSocket(t *testing.T) {
	skipOnWindows(t)
	_, err := BuildSeatbeltProfile([]string{"/usr/bin/true"}, policy.Policy{}, "relative.sock")
	if err == nil {
		t.Fatal("expected absolute socket error")
	}
}

func TestBuildSeatbeltProfileIncludesExeParent(t *testing.T) {
	skipOnWindows(t)
	exe := "/opt/custom/bin/server"
	profile, err := BuildSeatbeltProfile([]string{exe}, policy.Policy{}, "/tmp/warden/egress.sock")
	if err != nil {
		t.Fatal(err)
	}
	parent := filepath.Dir(exe)
	want := `(allow file-read* (subpath "` + parent + `"))`
	if !strings.Contains(profile, want) {
		t.Fatalf("profile missing exe parent grant %q\n%s", want, profile)
	}
	wantMap := `(allow file-map-executable (subpath "` + parent + `"))`
	if !strings.Contains(profile, wantMap) {
		t.Fatalf("profile missing exe parent map grant %q\n%s", wantMap, profile)
	}
}
