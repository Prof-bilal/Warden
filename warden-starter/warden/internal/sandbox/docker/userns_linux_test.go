//go:build linux

package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMapCurrentID(t *testing.T) {
	dir := t.TempDir()
	writeMap := func(name, content string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		return path
	}

	identity := writeMap("identity", "0 0 4294967295\n")
	if got, err := mapCurrentID(1000, identity); err != nil || got != 1000 {
		t.Fatalf("identity map: got (%d, %v), want (1000, nil)", got, err)
	}

	userns := writeMap("userns", "0 1000 1\n")
	if got, err := mapCurrentID(0, userns); err != nil || got != 1000 {
		t.Fatalf("userns map: got (%d, %v), want (1000, nil)", got, err)
	}
	if got, err := mapCurrentID(1, userns); err == nil {
		t.Fatalf("unmapped id: got (%d, nil), want error", got)
	}

	if _, err := mapCurrentID(0, filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing map file: got nil error, want error")
	}

	malformed := writeMap("malformed", "0 1000\n")
	if _, err := mapCurrentID(0, malformed); err == nil {
		t.Fatal("malformed map: got nil error, want error")
	}
}
