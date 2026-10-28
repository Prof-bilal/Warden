//go:build darwin || windows

package agentworkspace

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

func TestPortableProjectionPreservesProtectedValues(t *testing.T) {
	root := portableTestWorkspace(t)
	path := filepath.Join(root, "config.env")
	if err := os.WriteFile(path, []byte("API_KEY=hidden-value\nDEBUG=false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	p := policy.Policy{
		Filesystem: policy.Filesystem{Write: []string{root}},
		Agent: &policy.Agent{
			Workspace: root,
			Redaction: policy.Redaction{Files: []policy.RedactedFile{{
				Path: "config.env", Format: "dotenv", Fields: []string{"API_KEY"},
			}}},
		},
	}
	s, err := Prepare(p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	projected := filepath.Join(s.Dir, "config.env")
	b, err := os.ReadFile(projected)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b, []byte("hidden-value")) || !bytes.Contains(b, []byte("__WARDEN_REDACTED_")) {
		t.Fatal("portable projection did not redact the protected value")
	}
	if err = os.WriteFile(projected, bytes.ReplaceAll(b, []byte("DEBUG=false"), []byte("DEBUG=true")), 0600); err != nil {
		t.Fatal(err)
	}
	changes, err := s.Changes()
	if err != nil {
		t.Fatal(err)
	}
	var review bytes.Buffer
	if err = Review(&review, changes); err != nil || strings.Contains(review.String(), "hidden-value") {
		t.Fatalf("unsafe review output: %v %q", err, review.String())
	}
	if err = s.Apply(changes); err != nil {
		t.Fatal(err)
	}
	host, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(host, []byte("API_KEY=hidden-value")) || !bytes.Contains(host, []byte("DEBUG=true")) {
		t.Fatalf("host merge was incorrect: %q", host)
	}
}

func TestPortableProjectionRejectsSymlinks(t *testing.T) {
	root := portableTestWorkspace(t)
	if err := os.WriteFile(filepath.Join(root, "real"), []byte("value"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	p := policy.Policy{Filesystem: policy.Filesystem{Write: []string{root}}, Agent: &policy.Agent{Workspace: root}}
	if s, err := Prepare(p); err == nil {
		s.Close()
		t.Fatal("symlink was copied into portable projection")
	}
}

// portableTestWorkspace creates the fixture below the repository working
// directory instead of os.TempDir. Some hosted Windows runners expose their
// temporary directory through a symlinked ancestor; production correctly
// rejects such workspaces, but that would prevent this test from reaching the
// projection behavior it is meant to exercise.
func portableTestWorkspace(t *testing.T) string {
	t.Helper()
	base, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(base, ".warden-agent-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	return root
}
