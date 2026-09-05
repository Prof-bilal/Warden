package darwin

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

func TestBuildSeatbeltProfileDenyDefaultAndGrants(t *testing.T) {
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
		`(allow network-outbound (remote ip "127.0.0.1:18080"))`,
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

func TestBuildSeatbeltProfileRejectsConflicts(t *testing.T) {
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
	_, err := BuildSeatbeltProfile([]string{"/usr/bin/true"}, policy.Policy{}, "relative.sock")
	if err == nil {
		t.Fatal("expected absolute socket error")
	}
}

func TestBuildSeatbeltProfileIncludesExeParent(t *testing.T) {
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
}
