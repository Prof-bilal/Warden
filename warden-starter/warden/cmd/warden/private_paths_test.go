package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

func TestPrivateStateCannotOverlapResolvedGrantsOrExecutableDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix symlink fixture")
	}
	root := t.TempDir()
	public := filepath.Join(root, "public")
	if e := os.Mkdir(public, 0700); e != nil {
		t.Fatal(e)
	}
	alias := filepath.Join(root, "alias")
	if e := os.Symlink(public, alias); e != nil {
		t.Fatal(e)
	}
	private := filepath.Join(root, "private", "new", "key")
	if e := outsideUpstream(private, policy.Policy{}, []string{filepath.Join(public, "server")}, "linux"); e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{filepath.Join(public, "new", "control"), filepath.Join(alias, "new", "journal")} {
		p := policy.Policy{Filesystem: policy.Filesystem{Read: []string{alias}}}
		if outsideUpstream(path, p, nil, "linux") == nil {
			t.Fatal("private state accepted inside a resolved grant")
		}
		if outsideUpstream(path, policy.Policy{}, []string{filepath.Join(public, "server")}, "linux") == nil {
			t.Fatal("private state accepted inside executable directory")
		}
	}
	if outsideUpstream("/usr/warden-private-key", policy.Policy{}, nil, "linux") == nil {
		t.Fatal("private key accepted inside runtime base")
	}
}
