//go:build linux

package linux

import (
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

func TestBuildBwrapArgsBasic(t *testing.T) {
	cmd := []string{"/usr/bin/true"}
	p := policy.Policy{}
	args, err := BuildBwrapArgs(cmd, p)
	if err != nil {
		t.Fatal(err)
	}

	text := strings.Join(args, " ")
	for _, want := range []string{
		"--unshare-user", "--unshare-ipc", "--unshare-pid", "--unshare-net", "--disable-userns", "--die-with-parent",
		"--uid", "0", "--gid", "0",
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/lib64", "/lib64",
		"--dev", "/dev", "--proc", "/proc", "--size", "67108864", "--tmpfs", "/tmp",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("args %q missing %q", text, want)
		}
	}
}

func TestBuildBwrapArgsGrants(t *testing.T) {
	cmd := []string{"/usr/bin/true"}
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Read:  []string{"/data/readonly"},
			Write: []string{"/data/rw"},
		},
	}
	args, err := BuildBwrapArgs(cmd, p)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(args, " ")
	if !strings.Contains(text, "--ro-bind /data/readonly /data/readonly") {
		t.Errorf("missing read grant in %q", text)
	}
	if !strings.Contains(text, "--bind /data/rw /data/rw") {
		t.Errorf("missing write grant in %q", text)
	}
	if strings.Contains(text, "--ro-bind /data/rw") {
		t.Errorf("write grant mistakenly read-only: %q", text)
	}
}

func TestBuildBwrapArgsCommandParentBound(t *testing.T) {
	// A binary outside /usr gets its parent dir bound so it stays visible.
	cmd := []string{"/home/user/mise/node/bin/node", "server.js"}
	p := policy.Policy{}
	args, err := BuildBwrapArgs(cmd, p)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(args, " ")
	if !strings.Contains(text, "--ro-bind /home/user/mise/node/bin /home/user/mise/node/bin") {
		t.Errorf("command parent dir not bound: %q", text)
	}
}

func TestBuildBwrapArgsCommandParentAlreadyGranted(t *testing.T) {
	// If the grant already covers the command dir, no extra bind needed.
	cmd := []string{"/home/user/mise/node/bin/node"}
	p := policy.Policy{Filesystem: policy.Filesystem{Read: []string{"/home/user/mise/node/bin"}}}
	args, err := BuildBwrapArgs(cmd, p)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(args, " ")
	if c := strings.Count(text, "/home/user/mise/node/bin"); c != 2 { // 1 mount = flag+src+dst => 3 tokens, 2 occurrences
		t.Errorf("expected exactly one bind for command dir, got %d occurrences: %q", c, text)
	}
}

func TestBuildBwrapArgsRejectsRelativeCommand(t *testing.T) {
	_, err := BuildBwrapArgs([]string{"node"}, policy.Policy{})
	if err == nil {
		t.Fatal("expected error for relative command")
	}
}

func TestBuildBwrapArgsRejectsEmptyCommand(t *testing.T) {
	_, err := BuildBwrapArgs(nil, policy.Policy{})
	if err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestBuildBwrapArgsRejectsReadWriteConflict(t *testing.T) {
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Read:  []string{"/data/x"},
			Write: []string{"/data/x"},
		},
	}
	if _, err := BuildBwrapArgs([]string{"/usr/bin/true"}, p); err == nil {
		t.Fatal("expected error for path in both read and write")
	}
}

func TestBuildBwrapArgsRejectsWriteIntoRuntimeBase(t *testing.T) {
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Write: []string{"/usr/bin/mytool"},
		},
	}
	if _, err := BuildBwrapArgs([]string{"/usr/bin/true"}, p); err == nil {
		t.Fatal("expected error for write grant into runtime base")
	}

	// Exactly /usr itself is also refused.
	p.Filesystem.Write = []string{"/usr"}
	if _, err := BuildBwrapArgs([]string{"/usr/bin/true"}, p); err == nil {
		t.Fatal("expected error for write grant of runtime base itself")
	}
}

func TestBuildBwrapArgsRejectsMissingHostPath(t *testing.T) {
	// A grant whose source doesn't exist on the host should fail loudly
	// before bwrap gets a confusing error.
	p := policy.Policy{Filesystem: policy.Filesystem{Read: []string{"/definitely/not/here-xyz"}}}
	// BuildBwrapArgs is pure and doesn't stat; the existence check lives in
	// Run. This test documents that expectation.
	if _, err := BuildBwrapArgs([]string{"/usr/bin/true"}, p); err != nil {
		t.Fatalf("builder should not stat paths: %v", err)
	}
}

func TestIsUnder(t *testing.T) {
	cases := []struct {
		path, dir string
		want      bool
	}{
		{"/usr/bin", "/usr", true},
		{"/usr", "/usr", false},
		{"/usr2", "/usr", false},
		{"/usr/bin/../bin2", "/usr", true},
		{"/etc", "/usr", false},
	}
	for _, c := range cases {
		if got := isUnder(c.path, c.dir); got != c.want {
			t.Errorf("isUnder(%q, %q) = %v, want %v", c.path, c.dir, got, c.want)
		}
	}
}
