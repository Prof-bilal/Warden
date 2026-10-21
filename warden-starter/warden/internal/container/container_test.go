package container

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
	"gopkg.in/yaml.v3"
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

func TestSeccompProfileOmitsDangerousSyscalls(t *testing.T) {
	p, err := generateSeccompProfile(testPolicy(), TranslateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(p.Syscalls[0].Names, ",")
	for _, name := range []string{"ptrace", "kexec_load", "perf_event_open", "unshare", "mount", "pivot_root", "chroot"} {
		if strings.Contains(joined, name) {
			t.Errorf("profile allows %s", name)
		}
	}
}

func TestSaveSeccompProfileWritesFile(t *testing.T) {
	p, err := generateSeccompProfile(testPolicy(), TranslateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "seccomp.json")
	if err := SaveSeccompProfile(*p, path); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(path); err != nil || info.Size() == 0 {
		t.Fatalf("profile not written: %v", err)
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
	if m.Docker.NetworkMode != "none" {
		t.Errorf("network mode = %q, want none (a plain docker run cannot enforce network.allow)", m.Docker.NetworkMode)
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
	p := testPolicy()
	p.Network.Allow = nil
	cmd, err := RenderDockerCommand(p, TranslateOptions{Image: "myapp:latest"})
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

func TestRenderDockerCommandRefusesNetworkAllowlist(t *testing.T) {
	_, err := RenderDockerCommand(testPolicy(), TranslateOptions{Image: "myapp:latest"})
	if err == nil {
		t.Fatal("expected an error: a plain docker run cannot enforce network.allow")
	}
	if !strings.Contains(err.Error(), "network.allow") {
		t.Errorf("error should mention network.allow, got: %v", err)
	}
}

func TestNetworkPolicyUsesIPBlockPeers(t *testing.T) {
	p := testPolicy()
	p.Network.Allow = []string{"203.0.113.10", "api.github.com"}
	m, err := GenerateKubernetesManifests(p, TranslateOptions{Image: "myapp:latest"})
	if err != nil {
		t.Fatal(err)
	}
	rules := m.NetworkPolicy.Spec.Egress
	if len(rules) != 1 || len(rules[0].To) != 1 {
		t.Fatalf("expected one egress rule with one ipBlock peer, got %#v", rules)
	}
	peer := rules[0].To[0]
	if peer.IpBlock == nil || peer.IpBlock.CIDR != "203.0.113.10/32" {
		t.Fatalf("expected ipBlock 203.0.113.10/32, got %#v", peer)
	}
}

func TestNetworkPolicyDeniesHostnameOnlyAllowlist(t *testing.T) {
	// testPolicy() allows only api.github.com, which a standard NetworkPolicy
	// cannot express. It must fall back to deny-all rather than ports-only.
	p := testPolicy()
	m, err := GenerateKubernetesManifests(p, TranslateOptions{Image: "myapp:latest"})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.NetworkPolicy.Spec.Egress) != 0 {
		t.Fatalf("expected deny-all egress for a hostname-only allowlist, got %#v", m.NetworkPolicy.Spec.Egress)
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
	if strings.Contains(out, "kind: SeccompProfile") {
		t.Fatal("rendered Kubernetes bundle must not require the optional SeccompProfile CRD")
	}
	if !strings.Contains(out, "podSelector:\n        matchLabels:") {
		t.Fatalf("NetworkPolicy selector must use matchLabels:\n%s", out)
	}

	decoder := yaml.NewDecoder(strings.NewReader(out))
	var kinds []string
	for {
		var document map[string]any
		if err := decoder.Decode(&document); err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("rendered YAML is not parseable: %v", err)
		}
		if len(document) > 0 {
			kind, ok := document["kind"].(string)
			if !ok {
				t.Fatalf("rendered document has no kind: %#v", document)
			}
			kinds = append(kinds, kind)
		}
	}
	if strings.Join(kinds, ",") != "Deployment,NetworkPolicy" {
		t.Fatalf("unexpected rendered Kubernetes kinds: %v", kinds)
	}
}
