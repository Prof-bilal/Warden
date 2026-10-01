package packs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

func TestAllProfilesValidateWithNarrowGrants(t *testing.T) {
	dir := t.TempDir()
	runtime := filepath.Join(dir, "runtime")
	data := filepath.Join(dir, "data")
	_ = os.Mkdir(runtime, 0700)
	_ = os.Mkdir(data, 0700)
	seen := map[string]bool{}
	for _, pack := range List() {
		if seen[pack.ID] || pack.Version == "" || pack.Upstream == "" || pack.Status != "candidate" {
			t.Fatalf("invalid metadata: %+v", pack)
		}
		seen[pack.ID] = true
		path := ""
		if pack.PathMode != "none" {
			path = data
		}
		b, e := Generate(pack.ID, path, runtime)
		if e != nil {
			t.Fatal(e)
		}
		file := filepath.Join(dir, pack.ID+".yaml")
		_ = os.WriteFile(file, b, 0600)
		p, e := policy.Load(file)
		if e != nil {
			t.Fatal(e)
		}
		for _, host := range p.Network.Allow {
			if host == "*" {
				t.Fatal("wildcard host")
			}
		}
		if policy.CoversFile(p, filepath.Join(dir, "secret")) {
			t.Fatal("outside data/runtime path readable")
		}
		if pack.PathMode != "write" && len(p.Filesystem.Write) != 0 {
			t.Fatal("unexpected write grant")
		}
	}
	if len(seen) != 6 {
		t.Fatal("expected six initial profiles")
	}
}

func TestRejectsBroadAndOverlappingPaths(t *testing.T) {
	dir := t.TempDir()
	runtime := filepath.Join(dir, "runtime")
	_ = os.Mkdir(runtime, 0700)
	home, _ := os.UserHomeDir()
	for _, path := range []string{"", string(filepath.Separator), home} {
		if _, e := Generate("filesystem", path, runtime); e == nil {
			t.Fatalf("accepted broad path %q", path)
		}
	}
	if _, e := Generate("memory", dir, runtime); e == nil {
		t.Fatal("writable storage contains runtime")
	}
	if _, e := Generate("github", dir, runtime); e == nil {
		t.Fatal("irrelevant data grant silently accepted")
	}
}
