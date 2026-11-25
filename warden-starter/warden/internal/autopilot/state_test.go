//go:build linux || darwin || windows

package autopilot

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

func TestAutopilotEnrollmentLifecycle(t *testing.T) {
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
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
	t.Run("symlink", func(t *testing.T) {
		if err := state.Symlink(program, enrollmentName(project)); err != nil {
			if runtime.GOOS == "windows" && errors.Is(err, syscall.Errno(1314)) {
				t.Skip("creating a symlink requires developer mode or elevated privileges")
			}
			t.Fatal(err)
		}
		if _, err := Load(state, project); err == nil {
			t.Fatal("loaded symlink")
		}
		if err := Revoke(state, project); err != nil {
			t.Fatal(err)
		}
	})
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
