package container

import (
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

// testPolicy returns a representative Warden policy exercising the main
// translation paths: read filesystem, writable /tmp, hostname network allow,
// env allow, and resource limits.
func testPolicy() policy.Policy {
	return policy.Policy{
		Command:    []string{"./app", "--serve"},
		Filesystem: policy.Filesystem{Read: []string{"/etc/ssl"}, Write: []string{"/tmp"}},
		Network:    policy.Network{Allow: []string{"api.github.com"}},
		Env:        policy.Env{Allow: []string{"GITHUB_TOKEN"}},
		Limits:     policy.Limits{MemoryMB: 512, TimeoutS: 60},
	}
}

func TestTranslatePolicyDocker(t *testing.T) {
	m, err := TranslatePolicy(testPolicy(), TranslateOptions{Image: "myapp:latest"})
	if err != nil {
		t.Fatal(err)
	}
	if !m.Docker.ReadOnly {
		t.Error("expected read-only root filesystem")
	}
	if m.Docker.NetworkMode != "bridge" {
		t.Errorf("network mode = %q, want bridge", m.Docker.NetworkMode)
	}
	if len(m.Docker.CapDrop) != 1 || m.Docker.CapDrop[0] != "ALL" {
		t.Errorf("cap drop = %v, want [ALL]", m.Docker.CapDrop)
	}
	if m.Docker.Memory != "512m" {
		t.Errorf("memory = %q, want 512m", m.Docker.Memory)
	}
	foundTmpfs := false
	for _, tmp := range m.Docker.Tmpfs {
		if strings.HasPrefix(tmp, "/tmp") {
			foundTmpfs = true
		}
	}
	if !foundTmpfs {
		t.Errorf("expected tmpfs mount for /tmp, got %v", m.Docker.Tmpfs)
	}
}

func TestRenderDockerCommand(t *testing.T) {
	cmd, err := RenderDockerCommand(testPolicy(), TranslateOptions{Image: "myapp:latest"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd) < 3 || cmd[0] != "docker" || cmd[1] != "run" {
		t.Errorf("command does not start with docker run: %v", cmd)
	}
	joined := strings.Join(cmd, " ")
	for _, want := range []string{"--read-only", "--cap-drop ALL", "myapp:latest", "./app", "--serve"} {
		if !strings.Contains(joined, want) {
			t.Errorf("docker command missing %q: %s", want, joined)
		}
	}
}

func TestValidateKubernetesPolicyWarnings(t *testing.T) {
	// Hostname-based network rules surface an FQDN-limit warning.
	warnings := ValidateKubernetesPolicy(testPolicy())
	if len(warnings) == 0 {
		t.Fatal("expected FQDN warning for hostname network allow")
	}
	if !strings.Contains(strings.Join(warnings, "\n"), "hostnames") {
		t.Errorf("expected hostname warning, got %v", warnings)
	}

	// A deny-network policy with no funky paths/env should be clean.
	clean := testPolicy()
	clean.Network.Allow = nil
	if ws := ValidateKubernetesPolicy(clean); len(ws) != 0 {
		t.Errorf("expected no warnings for clean policy, got %v", ws)
	}
}

func TestGenerateKubernetesManifests(t *testing.T) {
	p := testPolicy()
	manifests, err := GenerateKubernetesManifests(p, TranslateOptions{Image: "myapp:latest", Namespace: "prod"})
	if err != nil {
		t.Fatal(err)
	}
	if manifests.Deployment == nil {
		t.Fatal("expected a Deployment manifest")
	}
	out, err := RenderKubernetesYAML(manifests)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"kind: Deployment",
		"namespace: prod",
		"readOnlyRootFilesystem: true",
		"kind: NetworkPolicy",
		"RuntimeDefault",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered YAML missing %q\n%s", want, out)
		}
	}
}
