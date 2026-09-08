package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/warden-sandbox/warden/internal/audit"
)

func TestFirstRunMarker(t *testing.T) {
	// Use isolated XDG_STATE_HOME
	tmp := t.TempDir()
	orig := os.Getenv("XDG_STATE_HOME")
	t.Cleanup(func() { os.Setenv("XDG_STATE_HOME", orig) })
	os.Setenv("XDG_STATE_HOME", tmp)
	// Ensure CI false and no opt-out, but also force TTY via env to avoid TTY check?
	// IsFirstRun checks TTY; in tests stdout/stderr are pipes (not TTY), so it would return false
	// unless we set WARDEN_FORCE_FIRST_RUN. For marker logic we test directly via file existence,
	// not IsFirstRun TTY gate. Instead test IsFirstRun with force env.
	os.Setenv("WARDEN_FORCE_FIRST_RUN", "1")
	t.Cleanup(func() { os.Unsetenv("WARDEN_FORCE_FIRST_RUN") })
	origCI := os.Getenv("CI")
	origNoFirst := os.Getenv("WARDEN_NO_FIRST_RUN")
	t.Cleanup(func() {
		os.Setenv("CI", origCI)
		if origNoFirst == "" {
			os.Unsetenv("WARDEN_NO_FIRST_RUN")
		} else {
			os.Setenv("WARDEN_NO_FIRST_RUN", origNoFirst)
		}
	})
	os.Unsetenv("CI")
	os.Unsetenv("WARDEN_NO_FIRST_RUN")

	if !IsFirstRun() {
		t.Fatal("IsFirstRun should be true before marker")
	}
	if err := MarkFirstRun(); err != nil {
		t.Fatalf("MarkFirstRun: %v", err)
	}
	if IsFirstRun() {
		t.Fatal("IsFirstRun should be false after MarkFirstRun")
	}
	// Idempotent
	if err := MarkFirstRun(); err != nil {
		t.Fatalf("second MarkFirstRun should be idempotent: %v", err)
	}
	// Verify marker file exists with correct perms
	dir, _ := audit.StateDir()
	marker := filepath.Join(dir, "welcomed")
	fi, err := os.Stat(marker)
	if err != nil {
		t.Fatalf("marker not found: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("marker perms = %o, want 0600", fi.Mode().Perm())
	}
}

func TestFirstRunDisabledInCI(t *testing.T) {
	tmp := t.TempDir()
	origXDG := os.Getenv("XDG_STATE_HOME")
	origCI := os.Getenv("CI")
	t.Cleanup(func() {
		os.Setenv("XDG_STATE_HOME", origXDG)
		os.Setenv("CI", origCI)
	})
	os.Setenv("XDG_STATE_HOME", tmp)
	os.Setenv("CI", "true")
	// Even with force, CI should disable
	if IsFirstRun() {
		t.Error("IsFirstRun should be false in CI")
	}
}

func TestFirstRunDisabledViaEnv(t *testing.T) {
	tmp := t.TempDir()
	origXDG := os.Getenv("XDG_STATE_HOME")
	origEnv := os.Getenv("WARDEN_NO_FIRST_RUN")
	t.Cleanup(func() {
		os.Setenv("XDG_STATE_HOME", origXDG)
		if origEnv == "" {
			os.Unsetenv("WARDEN_NO_FIRST_RUN")
		} else {
			os.Setenv("WARDEN_NO_FIRST_RUN", origEnv)
		}
	})
	os.Setenv("XDG_STATE_HOME", tmp)
	os.Setenv("WARDEN_NO_FIRST_RUN", "1")
	os.Setenv("WARDEN_FORCE_FIRST_RUN", "1")
	t.Cleanup(func() { os.Unsetenv("WARDEN_FORCE_FIRST_RUN") })
	if IsFirstRun() {
		t.Error("IsFirstRun should be false when WARDEN_NO_FIRST_RUN=1")
	}
}
