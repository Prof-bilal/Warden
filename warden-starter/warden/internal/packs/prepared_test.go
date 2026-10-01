package packs

import (
	"os"
	"os/exec"
	"path/filepath"
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
	_ = os.WriteFile(metadata, []byte(`{"name":"@modelcontextprotocol/server-filesystem","version":"2026.8.31"}`), 0600)
	argv, e := PreparedCommand("filesystem", dir, dir)
	if e != nil {
		t.Fatal(e)
	}
	if !filepath.IsAbs(argv[0]) || len(argv) != 3 || !strings.HasSuffix(argv[1], "index.js") {
		t.Fatalf("invalid prepared argv: %v", argv)
	}
}
