package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/sandbox"
	"gopkg.in/yaml.v3"
)

func TestRunDryRunPrintsPlan(t *testing.T) {
	backend, err := sandbox.Resolve("auto")
	if err != nil {
		t.Skipf("no sandbox backend available: %v", err)
	}
	bin := buildWarden(t)
	dir := t.TempDir()
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("filesystem: {read: [./data]}\n")
	if err := os.WriteFile(filepath.Join(dir, "warden.yaml"), body, 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "run", "--dry-run", "--backend", fmt.Sprint(backend), "--", bin, "version")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "XDG_STATE_HOME="+dir)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	var plan struct {
		Backend string `yaml:"backend"`
		Digest  string `yaml:"policy_sha256"`
		Policy  struct {
			Command    []string `yaml:"command"`
			Filesystem struct {
				Read []string `yaml:"read"`
			} `yaml:"filesystem"`
		} `yaml:"policy"`
	}
	if err := yaml.Unmarshal(out, &plan); err != nil {
		t.Fatalf("decode dry-run plan: %v\n%s", err, out)
	}
	if plan.Backend != fmt.Sprint(backend) || plan.Digest != fmt.Sprintf("%x", sha256.Sum256(body)) {
		t.Fatalf("incorrect backend or digest in %s", out)
	}
	expectedRead := filepath.Join(realDir, "data")
	if runtime.GOOS == "windows" {
		// Windows may expose the same temp tree through its 8.3 short alias.
		expectedRead = filepath.Join(dir, "data")
	}
	if !reflect.DeepEqual(plan.Policy.Command, []string{bin, "version"}) ||
		!reflect.DeepEqual(plan.Policy.Filesystem.Read, []string{expectedRead}) {
		t.Fatalf("incorrect effective policy in %s", out)
	}
}

func TestWizardPolicyPinsCommandWithoutGrants(t *testing.T) {
	dir := t.TempDir()
	cmd := []string{filepath.Join(dir, "test-launcher"), "argument with spaces", "--approve"}
	body, digest := wizardPolicy(cmd)
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	p, _, _, err := resolvePolicy(policySource{path: path, pin: digest})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cmd, p.Command) {
		t.Fatalf("command = %q, want %q", p.Command, cmd)
	}
	if len(p.Filesystem.Read)+len(p.Filesystem.Write)+len(p.Network.Allow)+len(p.Env.Allow) != 0 {
		t.Fatalf("starter policy includes grants: %#v", p)
	}
}

func TestResolveRunPackRejectsUnknownOrUnpinnedLauncher(t *testing.T) {
	for _, command := range [][]string{
		{"node", "@modelcontextprotocol/server-filesystem@2.3.0"},
		{"npx", "@modelcontextprotocol/server-filesystem@2026.8.31"},
		{"npx", "@modelcontextprotocol/server-filesystem@^2.3.0"},
		{"npx", "@modelcontextprotocol/server-filesystem@latest"},
		{"npx", "@modelcontextprotocol/server-memory"},
		{"npx", "@example/unknown-tool@1.0.0"},
	} {
		path, digest, err := resolveRunPack(command, "auto")
		if path != "" || digest != "" || !errors.Is(err, errNoRunPolicy) {
			t.Errorf("resolveRunPack(%v) = %q, %q, %v; want no match", command, path, digest, err)
		}
	}
}

func TestResolveRunPackRequiresExplicitPolicyForWrites(t *testing.T) {
	path, digest, err := resolveRunPack([]string{"npx", "@modelcontextprotocol/server-memory@2.2.1"}, "auto")
	if path != "" || digest != "" || err == nil || !strings.Contains(err.Error(), "write access requires explicit policy review") {
		t.Fatalf("resolveRunPack = %q, %q, %v; want explicit policy error", path, digest, err)
	}
}

func TestRunPackUpgradeRequiresReviewBeforeLaunch(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node.js unavailable")
	}
	bin := buildWarden(t)
	dir := t.TempDir()
	const artifact = "@modelcontextprotocol/server-filesystem"
	parent := filepath.Join(dir, "warden", "runtimes", "filesystem")
	previous := filepath.Join(parent, "2.2.0")
	current := filepath.Join(parent, "2.3.0")
	pkg := filepath.Join(current, "node_modules", filepath.FromSlash(artifact))
	shared := filepath.Join(dir, "shared")
	priorData := filepath.Join(dir, "prior-data")
	for _, path := range []string{previous, filepath.Join(pkg, "dist"), shared, priorData} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	old := fmt.Appendf(nil, "filesystem: {read: [%q], protect_secrets: true}\n", filepath.ToSlash(priorData))
	pin := fmt.Sprintf("%x", sha256.Sum256(old))
	for name, body := range map[string][]byte{
		filepath.Join(previous, "run-"+pin+".yaml"): old,
		filepath.Join(pkg, "package.json"):          []byte(`{"name":"@modelcontextprotocol/server-filesystem","version":"2.3.0"}`),
		filepath.Join(pkg, "dist", "index.js"):      []byte("process.exit(99);"),
	} {
		if err := os.WriteFile(name, body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(bin, "run", "npx", artifact, shared)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader("")
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dir, "XDG_STATE_HOME="+dir, "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	if status, ok := err.(*exec.ExitError); !ok || status.ExitCode() != 2 ||
		!strings.Contains(string(out), "Pack upgrade requires an interactive review") ||
		!strings.Contains(string(out), "prior-data") || !strings.Contains(string(out), "shared") {
		t.Fatalf("expected a pack diff and refusal before launch, got %v\n%s", err, out)
	}
	policies, err := filepath.Glob(filepath.Join(current, "run-*.yaml"))
	if err != nil || len(policies) != 0 {
		t.Fatalf("upgrade persisted an unreviewed policy: %v, %v", policies, err)
	}
}

func TestRunPolicySelection(t *testing.T) {
	bin := buildWarden(t)
	dir := t.TempDir()
	explicit := filepath.Join(dir, "explicit.yaml")
	local := filepath.Join(dir, "warden.yaml")
	if err := os.WriteFile(explicit, []byte("filesystem: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}

	run := func(want string, args ...string) {
		t.Helper()
		const launcher = "warden-test-launcher-that-does-not-exist"
		cmd := exec.Command(bin, append(append([]string{"run"}, args...), launcher, "--approve")...)
		cmd.Dir = dir
		cmd.Stdin = strings.NewReader("")
		cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+dir, "NO_COLOR=1")
		out, err := cmd.CombinedOutput()
		if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 2 || !strings.Contains(string(out), want) {
			t.Fatalf("run %v = %v, %s; want exit 2 containing %q", args, err, out, want)
		}
	}
	run("no policy found in the working directory")
	if err := os.WriteFile(local, []byte("invalid_policy_field: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("parse policy \"warden.yaml\"")
	run("not found on PATH", "--policy", explicit)
	if err := os.WriteFile(local, []byte("filesystem: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("not found on PATH")
}

func TestResolvePolicy(t *testing.T) {
	data := []byte("filesystem: {read: [./data]}\n")
	dir := t.TempDir()
	path := filepath.Join(dir, "warden.yaml")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	wantDigest := fmt.Sprintf("%x", sha256.Sum256(data))
	for _, pin := range []string{"", wantDigest} {
		p, digest, source, err := resolvePolicy(policySource{path: path, pin: pin})
		if err != nil {
			t.Fatal(err)
		}
		if digest != wantDigest || source != "file" {
			t.Fatalf("got digest %q and source %q, want %q and file", digest, source, wantDigest)
		}
		if len(p.Filesystem.Read) != 1 || p.Filesystem.Read[0] != filepath.Join(dir, "data") {
			t.Fatalf("read grants = %v", p.Filesystem.Read)
		}
	}

	if _, _, _, err := resolvePolicy(policySource{path: path, pin: strings.Repeat("0", 64)}); err == nil || !strings.Contains(err.Error(), "policy changed since wrapping") {
		t.Fatalf("mismatched digest error = %v", err)
	}
	_, digest, source, err := resolvePolicy(policySource{path: path, generated: data, provenance: "learned"})
	if err != nil || digest != wantDigest || source != "learned" {
		t.Fatalf("generated policy digest = %q, source = %q, error = %v", digest, source, err)
	}
}
