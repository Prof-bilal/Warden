//go:build linux

package autopilot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/warden-sandbox/warden/internal/agentworkspace"
)

func TestAutopilotPolicyKeepsProjectDataInProjection(t *testing.T) {
	project := t.TempDir()
	for name, data := range map[string]string{
		"package.json": `{"scripts":{"build":"untrusted project script"}}`,
		".mcp.json":    `{"untrusted":"tool config"}`,
		".ENV":         "TOKEN=disposable",
	} {
		if err := os.WriteFile(filepath.Join(project, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(project, "package.json"), filepath.Join(project, "go.mod")); err != nil {
		t.Fatal(err)
	}
	hints, err := Inspect(project)
	if err != nil || len(hints.Hints) != 2 {
		t.Fatalf("inspection did not use regular markers only: %+v, %v", hints, err)
	}
	if err := os.Remove(filepath.Join(project, "go.mod")); err != nil {
		t.Fatal(err)
	}
	program, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Enroll(project, program, "grant-all"); err == nil {
		t.Fatal("unsupported mode accepted")
	}
	e, err := Enroll(project, program, ReviewedEdits)
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.Policy(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Env.Allow) != 0 || len(p.Network.Allow) != 0 || len(p.TemporaryGrants) != 0 || p.MCP != nil || len(p.Filesystem.Write) != 1 || p.Filesystem.Write[0] != project {
		t.Fatalf("unexpected grants: %+v", p)
	}
	s, err := agentworkspace.Prepare(p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, name := range []string{".ENV", ".mcp.json"} {
		if _, err := os.Stat(filepath.Join(s.Dir, name)); !os.IsNotExist(err) {
			t.Fatalf("excluded file %q projected: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(s.Dir, ".mcp.json"), []byte("candidate"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Changes(); err == nil {
		t.Fatal("excluded candidate path reached review")
	}
}
