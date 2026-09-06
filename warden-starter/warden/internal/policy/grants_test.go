package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProposeFileGrant(t *testing.T) {
	grant, write, ok := ProposeFileGrant("openat", "/srv/data/file.txt")
	if !ok || grant != filepath.FromSlash("/srv/data/file.txt") || write {
		t.Fatalf("read = %q, %v, %v", grant, write, ok)
	}
	grant, write, ok = ProposeFileGrant("creat", "/srv/data/new.txt")
	if !ok || grant != filepath.FromSlash("/srv/data") || !write {
		t.Fatalf("creat = %q, %v, %v", grant, write, ok)
	}
	for _, tc := range []struct{ action, resource string }{
		{"openat", "relative/path"},
		{"openat", "/proc/self/cmdline"},
		{"openat", "/usr/lib/libc.so"},
	} {
		if _, _, ok := ProposeFileGrant(tc.action, tc.resource); ok {
			t.Errorf("ProposeFileGrant(%q, %q) should not propose", tc.action, tc.resource)
		}
	}
}

func TestCoversFile(t *testing.T) {
	p := Policy{Filesystem: Filesystem{Read: []string{filepath.FromSlash("/srv/ro")}, Write: []string{filepath.FromSlash("/srv/rw")}}}
	for _, path := range []string{
		filepath.FromSlash("/srv/ro"),
		filepath.FromSlash("/srv/ro/a.txt"),
		filepath.FromSlash("/srv/rw"),
		filepath.FromSlash("/srv/rw/sub/b.txt"),
	} {
		if !CoversFile(p, path) {
			t.Errorf("CoversFile(%q) = false, want true", path)
		}
	}
	for _, path := range []string{
		filepath.FromSlash("/srv/other"),
		filepath.FromSlash("/etc/passwd"),
	} {
		if CoversFile(p, path) {
			t.Errorf("CoversFile(%q) = true, want false", path)
		}
	}
}

func TestAddHost(t *testing.T) {
	p := Policy{}
	added, err := p.AddHost("API.Example.com:443")
	if err != nil || !added {
		t.Fatalf("AddHost = %v, %v", added, err)
	}
	if len(p.Network.Allow) != 1 || p.Network.Allow[0] != "api.example.com" {
		t.Fatalf("allow = %v", p.Network.Allow)
	}
	added, err = p.AddHost("api.example.com")
	if err != nil || added {
		t.Fatalf("duplicate AddHost = %v, %v", added, err)
	}
	if _, err := p.AddHost("not a host!!"); err == nil {
		t.Fatal("invalid host must fail")
	}
}

func TestAddFileGrantSubsumes(t *testing.T) {
	p := Policy{}
	if _, err := p.AddFileGrant(filepath.FromSlash("/srv/data/file.txt"), false); err != nil {
		t.Fatal(err)
	}
	added, err := p.AddFileGrant(filepath.FromSlash("/srv/data"), true)
	if err != nil || !added {
		t.Fatalf("AddFileGrant = %v, %v", added, err)
	}
	if len(p.Filesystem.Read) != 0 {
		t.Fatalf("write grant should subsume read, got %v", p.Filesystem.Read)
	}
	added, err = p.AddFileGrant(filepath.FromSlash("/srv/data/other.txt"), false)
	if err != nil || added {
		t.Fatalf("covered read = %v, %v", added, err)
	}
	if _, err := p.AddFileGrant("relative", false); err == nil {
		t.Fatal("relative grant must fail")
	}
}

func TestSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.yaml")
	p := Policy{Filesystem: Filesystem{Read: []string{filepath.FromSlash("/srv/ro")}}}
	if err := p.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Filesystem.Read) != 1 || loaded.Filesystem.Read[0] != filepath.FromSlash("/srv/ro") {
		t.Fatalf("round trip = %+v", loaded.Filesystem)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
}

func TestNormalizeDropsShadowedReadsSortsAndDedups(t *testing.T) {
	p := Policy{Filesystem: Filesystem{
		Read:  []string{filepath.FromSlash("/srv/b"), filepath.FromSlash("/srv/a"), filepath.FromSlash("/srv/a"), filepath.FromSlash("/srv/w/file.txt"), filepath.FromSlash("/srv/keep")},
		Write: []string{filepath.FromSlash("/srv/w")},
	}}
	p.Normalize()
	wantRead := []string{filepath.FromSlash("/srv/a"), filepath.FromSlash("/srv/b"), filepath.FromSlash("/srv/keep")}
	if len(p.Filesystem.Read) != len(wantRead) {
		t.Fatalf("Read = %v, want %v", p.Filesystem.Read, wantRead)
	}
	for i := range wantRead {
		if p.Filesystem.Read[i] != wantRead[i] {
			t.Fatalf("Read = %v, want %v", p.Filesystem.Read, wantRead)
		}
	}
	if len(p.Filesystem.Write) != 1 || p.Filesystem.Write[0] != filepath.FromSlash("/srv/w") {
		t.Fatalf("Write = %v, want [%s]", p.Filesystem.Write, filepath.FromSlash("/srv/w"))
	}
}

func TestResolveExecutable(t *testing.T) {
	got, err := ResolveExecutable([]string{filepath.FromSlash("/usr/bin/node"), "server.js"})
	if err != nil || got[0] != filepath.FromSlash("/usr/bin/node") {
		t.Fatalf("absolute passthrough = %v, %v", got, err)
	}
	if _, err := ResolveExecutable(nil); err == nil {
		t.Fatal("empty command must fail closed")
	}
	if _, err := ResolveExecutable([]string{"definitely-not-a-warden-test-binary-xyz"}); err == nil {
		t.Fatal("unresolvable bare name must fail closed")
	}
}
