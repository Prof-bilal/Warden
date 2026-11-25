package packs

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPreparedCommandRequiresRecordedVersion(t *testing.T) {
	if _, e := exec.LookPath("node"); e != nil {
		t.Skip("Node.js unavailable")
	}
	dir := t.TempDir()
	root := filepath.Join(dir, "node_modules", "@modelcontextprotocol", "server-filesystem")
	_ = os.MkdirAll(filepath.Join(root, "dist"), 0700)
	metadata := filepath.Join(root, "package.json")
	_ = os.WriteFile(metadata, []byte(`{"name":"@modelcontextprotocol/server-filesystem","version":"wrong"}`), 0600)
	_ = os.WriteFile(filepath.Join(root, "dist", "index.js"), []byte(""), 0600)
	if _, e := PreparedCommand("filesystem", dir, dir); e == nil {
		t.Fatal("wrong installed version accepted")
	}
	pack, e := Find("filesystem")
	if e != nil {
		t.Fatal(e)
	}
	_ = os.WriteFile(metadata, []byte(`{"name":"@modelcontextprotocol/server-filesystem","version":"`+pack.Version+`"}`), 0600)
	argv, e := PreparedCommand("filesystem", dir, dir)
	if e != nil {
		t.Fatal(e)
	}
	if !filepath.IsAbs(argv[0]) || len(argv) < 3 || !strings.HasSuffix(argv[len(argv)-2], "index.js") {
		t.Fatalf("invalid prepared argv: %v", argv)
	}
	if runtime.GOOS == "windows" {
		if len(argv) != 5 || argv[1] != "--import" || !strings.HasPrefix(argv[2], "data:text/javascript;base64,") {
			t.Fatalf("Windows prepared command lacks the realpath compatibility preload: %v", argv)
		}
	} else if len(argv) != 3 {
		t.Fatalf("invalid prepared argv: %v", argv)
	}
}
