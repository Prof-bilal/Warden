package docker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

func TestBuildDockerArgsDenyByDefaultMounts(t *testing.T) {
	readDir := t.TempDir()
	writeDir := t.TempDir()
	bridge := filepath.Join(t.TempDir(), "warden")
	if err := os.WriteFile(bridge, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	sockDir := t.TempDir()
	cmd := []string{"/usr/bin/true"}
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Read:  []string{readDir},
			Write: []string{writeDir},
		},
		Limits: policy.Limits{MemoryMB: 256},
	}
	args, err := BuildDockerArgs(cmd, p, bridge, sockDir, "alpine:3.20")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"--network none",
		"--read-only",
		"--memory 256m",
		readDir + ":" + readDir + ":ro",
		writeDir + ":" + writeDir + ":rw",
		bridge + ":/.warden/proxy-bridge:ro",
		sockDir + ":/.warden/host-proxy:ro",
		"alpine:3.20",
		"__proxy-bridge",
		"/usr/bin/true",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("docker args missing %q\nargs: %v", want, args)
		}
	}
}

func TestBuildDockerArgsRejectsReadWriteConflict(t *testing.T) {
	dir := t.TempDir()
	bridge := filepath.Join(t.TempDir(), "warden")
	_ = os.WriteFile(bridge, []byte("x"), 0o755)
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Read:  []string{dir},
			Write: []string{dir},
		},
	}
	_, err := BuildDockerArgs([]string{"/usr/bin/true"}, p, bridge, t.TempDir(), "alpine:3.20")
	if err == nil {
		t.Fatal("expected read/write conflict error")
	}
}

func TestBuildDockerArgsRequiresAbsoluteCommand(t *testing.T) {
	bridge := filepath.Join(t.TempDir(), "warden")
	_ = os.WriteFile(bridge, []byte("x"), 0o755)
	_, err := BuildDockerArgs([]string{"true"}, policy.Policy{}, bridge, t.TempDir(), "alpine:3.20")
	if err == nil {
		t.Fatal("expected absolute path error")
	}
}

func TestInjectEnvFlags(t *testing.T) {
	got := injectEnvFlags([]string{"run", "--rm", "alpine", "/bin/true"}, []string{"FOO=bar"}, []string{"HTTP_PROXY=http://127.0.0.1:18080"})
	wantPrefix := []string{"run", "-e", "FOO=bar", "-e", "HTTP_PROXY=http://127.0.0.1:18080", "--rm", "alpine", "/bin/true"}
	if strings.Join(got, " ") != strings.Join(wantPrefix, " ") {
		t.Fatalf("injectEnvFlags = %v, want %v", got, wantPrefix)
	}
}

func TestAvailableDoesNotPanic(t *testing.T) {
	_ = Available()
}
