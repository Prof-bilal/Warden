package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHookCheckPreToolUse(t *testing.T) {
	bin := buildWarden(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "warden.yaml")
	if err := os.WriteFile(path, []byte("command: [git, status]\nfilesystem: {read: [./read], write: [./write], deny: [./read/private], protect_secrets: true}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		tool, field, value string
		code               int
	}{
		{"Bash", "command", "git status", 0},
		{"PowerShell", "command", "git status", 0},
		{"Bash", "command", "git diff", 2},
		{"Bash", "command", "git status\n git diff", 2},
		{"Read", "file_path", filepath.Join(dir, "read", "sample.txt"), 0},
		{"Read", "file_path", filepath.Join(dir, "read", "private", "sample.txt"), 2},
		{"Read", "file_path", filepath.Join(dir, "write", ".env"), 2},
		{"Write", "file_path", filepath.Join(dir, "write", "sample.txt"), 0},
		{"Edit", "file_path", filepath.Join(dir, "read", "sample.txt"), 2},
		{"Read", "file_path", "relative.txt", 2},
		{"Read", "wrong_field", filepath.Join(dir, "read", "sample.txt"), 2},
	} {
		t.Run(tc.tool+"/"+tc.value, func(t *testing.T) {
			payload, err := json.Marshal(map[string]any{
				"hook_event_name": "PreToolUse", "tool_name": tc.tool, "cwd": dir,
				"tool_use_id": "toolu_fixture", "tool_input": map[string]string{tc.field: tc.value},
			})
			if err != nil {
				t.Fatal(err)
			}
			checkHookExit(t, bin, path, payload, tc.code)
		})
	}
	checkHookExit(t, bin, path, []byte(`{"hook_event_name":"PreToolUse",`), 2)
}

func checkHookExit(t *testing.T, bin, policy string, payload []byte, code int) {
	t.Helper()
	cmd := exec.Command(bin, "hook", "check", "--policy", policy)
	cmd.Stdin = bytes.NewReader(payload)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	got := 0
	if err != nil {
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		got = exit.ExitCode()
	}
	if got != code || len(out) != 0 || (code == 2) != (stderr.Len() > 0) {
		t.Fatalf("hook output %q, stderr %q, exit %d, want %d", out, &stderr, got, code)
	}
}

func TestHookInstallUndoRestoresSettingsBytes(t *testing.T) {
	bin := buildWarden(t)
	dir := t.TempDir()
	settings := filepath.Join(dir, "settings.local.json")
	policy := filepath.Join(dir, "warden.yaml")
	before := []byte("{\n \"hooks\":{\"SessionStart\":[]}, \"permissions\": {\"deny\": [\"Read(/private/**)\"]}\n}\n")
	if err := os.WriteFile(settings, before, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policy, []byte("filesystem: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	env := []string{"WARDEN_SETUP_STATE_DIR=" + filepath.Join(dir, "undo")}
	install := []string{"hook", "install", "--policy", policy, "--config", settings}
	undo := []string{"hook", "uninstall", "--config", settings}
	if _, stderr, code := runWarden(t, bin, env, install...); code != 0 {
		t.Fatalf("install exited %d: %s", code, stderr)
	}
	after, err := os.ReadFile(settings)
	if err != nil || !bytes.Contains(after, []byte(`"onFailure": "block"`)) ||
		!bytes.Contains(after, []byte(`"SessionStart": []`)) || !bytes.Contains(after, []byte(`"Read(/private/**)"`)) {
		t.Fatalf("installed settings = %q, %v", after, err)
	}
	if _, stderr, code := runWarden(t, bin, env, install...); code != 0 {
		t.Fatalf("repeat install exited %d: %s", code, stderr)
	}
	if repeated, err := os.ReadFile(settings); err != nil || !bytes.Equal(after, repeated) {
		t.Fatalf("repeat install changed settings: %s, %v", repeated, err)
	}
	if err := os.WriteFile(settings, append(bytes.Clone(after), '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if _, stderr, code := runWarden(t, bin, env, undo...); code != 2 || !strings.Contains(stderr, "settings changed") {
		t.Fatalf("undo accepted settings drift, exit %d: %s", code, stderr)
	}
	if err := os.WriteFile(settings, after, 0600); err != nil {
		t.Fatal(err)
	}
	if _, stderr, code := runWarden(t, bin, env, undo...); code != 0 {
		t.Fatalf("undo exited %d: %s", code, stderr)
	}
	if restored, err := os.ReadFile(settings); err != nil || !bytes.Equal(restored, before) {
		t.Fatalf("restored settings = %q, %v", restored, err)
	}
}
