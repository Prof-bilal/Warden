package docker

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

func TestMain(m *testing.M) {
	if os.Getenv("WARDEN_DOCKER_BRIDGE") == "" && runtime.GOOS == "linux" {
		wd, _ := os.Getwd()
		bridgeSrc := filepath.Join(wd, "..", "..", "..", "cmd", "warden")
		if info, err := os.Stat(bridgeSrc); err == nil && info.IsDir() {
			dir, err := os.MkdirTemp("", "warden-test-bridge-*")
			if err == nil {
				bridge := filepath.Join(dir, "warden")
				cmd := exec.Command("go", "build", "-o", bridge, bridgeSrc)
				cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
				if _, err := cmd.CombinedOutput(); err == nil {
					os.Setenv("WARDEN_DOCKER_BRIDGE", bridge)
				}
			}
		}
	}
	os.Exit(m.Run())
}

// False-positive hardening (plan §5): requireDocker proves the backend can
// actually launch a target before any denial test runs, so a broken daemon or
// image can never be reported as a sandbox PASS.
const startupMarkerDocker = "WARDEN_DOCKER_UP"

func requireDocker(t *testing.T) {
	t.Helper()
	if !Available() {
		t.Skip("docker daemon not usableskipping Docker integration test")
	}
	image := Image()
	if err := exec.Command("docker", "image", "inspect", image).Run(); err != nil {
		t.Skipf("docker image %q not presentskipping", image)
	}

	// Positive control: the sandboxed target must really start and execute.
	// Run() passes stdio through but returns only the exit code, so prove
	// execution via a granted write the test can verify on the host. FAIL
	// (not skip) if nota skip here would mask a dead backend behind
	// green tests.
	//
	// Use a directory outside /tmp because the Docker backend provides /tmp
	// as an isolated tmpfs, and t.TempDir() returns paths under /tmp which
	// would conflict with that tmpfs mount.
	writeDir, err := os.MkdirTemp("/var/tmp", "warden-docker-test-*")
	if err != nil {
		t.Fatalf("create temp dir outside /tmp: %v", err)
	}
	// Docker bind-mounts preserve host UIDs. The container runs as root
	// (UID 0) but the temp dir is owned by the test process UID. chmod
	// 0777 so the container root can write into it.
	os.Chmod(writeDir, 0777)
	t.Cleanup(func() { os.RemoveAll(writeDir) })
	marker := filepath.Join(writeDir, "started.marker")
	p := policy.Policy{Filesystem: policy.Filesystem{Write: []string{writeDir}}}
	out, err := Run([]string{"/bin/sh", "-c", "echo " + startupMarkerDocker + " > " + marker}, p)
	if err != nil {
		t.Fatalf("positive control: docker sandbox cannot start a target: %v\noutput: %v", err, out)
	}
	data, readErr := os.ReadFile(marker)
	if readErr != nil {
		t.Fatalf("positive control: startup marker file missingtarget never really ran: %v", readErr)
	}
	if !strings.Contains(string(data), startupMarkerDocker) {
		t.Fatalf("positive control: marker file lacks startup marker: %q", string(data))
	}
}

func TestDockerBlocksUngrantedRead(t *testing.T) {
	requireDocker(t)

	secretDir, err := os.MkdirTemp("/var/tmp", "warden-docker-secret-*")
	if err != nil {
		t.Fatalf("create secret dir: %v", err)
	}
	os.Chmod(secretDir, 0777)
	t.Cleanup(func() { os.RemoveAll(secretDir) })
	if err := os.WriteFile(filepath.Join(secretDir, "secret.txt"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	scriptDir, err := os.MkdirTemp("/var/tmp", "warden-docker-script-*")
	if err != nil {
		t.Fatalf("create script dir: %v", err)
	}
	os.Chmod(scriptDir, 0777)
	t.Cleanup(func() { os.RemoveAll(scriptDir) })
	script := filepath.Join(scriptDir, "probe.sh")
	body := "#!/bin/sh\ncat \"$1\"\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{scriptDir}},
	}
	code, err := Run([]string{"/bin/sh", script, filepath.Join(secretDir, "secret.txt")}, p)
	if err != nil {
		// Fail-closed startup errors are acceptable; escape is what we assert.
		if strings.Contains(err.Error(), "docker") {
			t.Skipf("docker run failed to start: %v", err)
		}
		t.Fatalf("Run: %v", err)
	}
	if code == 0 {
		t.Fatal("expected non-zero exit when reading ungranted path")
	}
}
