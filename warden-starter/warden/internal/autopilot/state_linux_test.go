//go:build linux

package autopilot

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/warden-sandbox/warden/internal/agentworkspace"
)

func TestAutopilotEnrollmentLifecycle(t *testing.T) {
	project := t.TempDir()
	stateDir := filepath.Join(t.TempDir(), "state")
	if _, err := OpenState(filepath.Join(project, "state"), project, true); err == nil {
		t.Fatal("state inside project accepted")
	}
	state, err := OpenState(stateDir, project, true)
	if err != nil {
		t.Fatal(err)
	}
	defer state.Close()
	program, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	e, err := Enroll(project, program, ReadOnly)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Load(state, project); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing enrollment: %v", err)
	}
	if err := Save(state, e); err != nil {
		t.Fatal(err)
	}
	if err := Save(state, e); !errors.Is(err, os.ErrExist) {
		t.Fatalf("replaced an approval, or wrong error: %v", err)
	}
	got, err := Load(state, project)
	if err != nil || got != e {
		t.Fatalf("load = %+v, %v", got, err)
	}
	if _, err := got.Policy(t.TempDir()); err == nil {
		t.Fatal("approved policy accepted in another project")
	}
	if err := Revoke(state, project); err != nil {
		t.Fatal(err)
	}
	if err := state.Symlink(program, enrollmentName(project)); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(state, project); err == nil {
		t.Fatal("loaded symlink")
	}
	if err := Revoke(state, project); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"revision":1,"mode":"read-only","network":{"allow":["*"]}}`)
	if err := os.WriteFile(filepath.Join(stateDir, enrollmentName(project)), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(state, project); err == nil {
		t.Fatal("accepted fields outside enrollment schema")
	}
	if err := Revoke(state, project); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(state, project); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("revocation did not remove enrollment: %v", err)
	}
}

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
