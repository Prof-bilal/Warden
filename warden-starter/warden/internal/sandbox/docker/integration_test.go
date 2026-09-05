package docker

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

func requireDocker(t *testing.T) {
	t.Helper()
	if !Available() {
		t.Skip("docker daemon not usable — skipping Docker integration test")
	}
	image := Image()
	if err := exec.Command("docker", "image", "inspect", image).Run(); err != nil {
		t.Skipf("docker image %q not present — skipping", image)
	}
}

func TestDockerBlocksUngrantedRead(t *testing.T) {
	requireDocker(t)

	secretDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(secretDir, "secret.txt"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	scriptDir := t.TempDir()
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
