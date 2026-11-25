//go:build linux || darwin || windows

package main

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/autopilot"
)

func TestAutopilotNativeReadOnly(t *testing.T) {
	requireLinuxContainment(t)
	readme, stateDir, program := nativeAutopilotFixture(t)
	enrollment, err := autopilot.Enroll(filepath.Dir(readme), program, autopilot.ReadOnly)
	if err != nil {
		t.Fatal(err)
	}
	state, err := autopilot.OpenState(filepath.Join(stateDir, "autopilot"), enrollment.Project, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := autopilot.Save(state, enrollment); err != nil {
		state.Close()
		t.Fatal(err)
	}
	state.Close()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	address := listener.Addr().String()
	conn, err := net.DialTimeout("tcp4", address, time.Second)
	if err != nil {
		t.Fatalf("host network positive control: %v", err)
	}
	conn.Close()

	setupStart := time.Now()
	if code, err := cmdAutopilot([]string{"setup"}); err != nil || code != 0 {
		t.Fatalf("Autopilot setup proof: exit %d, %v", code, err)
	}
	setupTime := time.Since(setupStart)

	args := []string{"run", "--", "-test.run=^TestAutopilotNativeProbe$", "--", readme, address}
	launchStart := time.Now()
	if code, err := cmdAutopilot(args); err != nil || code != 0 {
		t.Fatalf("native Autopilot run: exit %d, %v", code, err)
	}
	launchTime := time.Since(launchStart)
	logs, err := os.ReadFile(filepath.Join(stateDir, "audit.jsonl"))
	if err != nil || !strings.Contains(string(logs), `"type":"autopilot","action":"launch"`) {
		t.Fatalf("audit log lacks Autopilot launch event: %v", err)
	}
	got, err := os.ReadFile(readme)
	if err != nil || string(got) != "disposable project" {
		t.Fatalf("original project contents changed: %q, %v", got, err)
	}
	if code, err := cmdAutopilot([]string{"revoke"}); err != nil || code != 0 {
		t.Fatalf("Autopilot revoke: exit %d, %v", code, err)
	}
	if code, err := cmdAutopilot(args); err == nil || code == 0 {
		t.Fatalf("launch after revocation: exit %d, %v", code, err)
	}
	t.Logf("AUTOPILOT_TIMING setup_ms=%.3f launch_ms=%.3f", float64(setupTime.Microseconds())/1000, float64(launchTime.Microseconds())/1000)
}

func nativeAutopilotFixture(t *testing.T) (readme, stateDir, program string) {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := os.MkdirTemp(home, "warden-autopilot-native-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(fixture) })
	fixture, err = filepath.EvalSymlinks(fixture)
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(fixture, "project with spaces")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	readme = filepath.Join(project, "README.md")
	if err := os.WriteFile(readme, []byte("disposable project"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".env"), []byte("TOKEN=disposable"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	t.Setenv("XDG_STATE_HOME", filepath.Join(fixture, "state"))
	stateDir, err = audit.StateDir()
	if err != nil {
		t.Fatal(err)
	}
	program, err = os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return readme, stateDir, program
}

func TestAutopilotNativeProbe(t *testing.T) {
	args := os.Args
	if len(args) < 2 || args[1] != "-test.run=^TestAutopilotNativeProbe$" {
		return // The parent test selects this probe inside the sandbox.
	}
	if len(args) != 5 || args[2] != "--" {
		t.Fatal("native probe operands missing")
	}
	content, err := os.ReadFile("README.md")
	if err != nil || string(content) != "disposable project" {
		t.Fatalf("projected read: %q, %v", content, err)
	}
	if err := os.WriteFile("README.md", []byte("must fail"), 0600); !blockedAutopilotWrite(err) {
		t.Fatalf("write existing file in read-only workspace: %v", err)
	}
	if err := os.WriteFile("new.txt", []byte("must fail"), 0600); !blockedAutopilotWrite(err) {
		t.Fatalf("create file in read-only workspace: %v", err)
	}
	if _, err := os.ReadFile(".env"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("secret file reached the projection: %v", err)
	}
	if _, err := os.ReadFile(args[3]); !errors.Is(err, os.ErrPermission) && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read original project outside the projection: %v", err)
	}
	if conn, err := net.DialTimeout("tcp4", args[4], time.Second); err == nil {
		conn.Close()
		t.Fatal("offline probe connected to the host listener")
	}
}

func blockedAutopilotWrite(err error) bool {
	return errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.EROFS)
}
