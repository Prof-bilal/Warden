package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/guard"
)

// The Linux sandbox re-executes the test binary as its proxy bridge.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "__proxy-bridge" {
		cmdProxyBridge(os.Args[2:])
		return
	}
	os.Exit(m.Run())
}

func TestApplyInstallGuardPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		userStrict bool
		wantAge    time.Duration
		wantError  string
	}{
		{"tighten", "version: 2\nguard: {minimum_release_age: 72h, strict: true}", false, 72 * time.Hour, ""},
		{"keep user rules", "version: 2\nguard: {minimum_release_age: 1h, strict: false}", true, 48 * time.Hour, ""},
		{"v1", "version: 1\nguard: {}", false, 0, "guard requires policy version 2"},
		{"negative", "version: 2\nguard: {minimum_release_age: -1h}", false, 0, "must be nonnegative"},
		{"script authorization", "version: 2\nguard: {script_allowlist: ['example@1.0.0']}", false, 0, "field script_allowlist not found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "warden.yaml")
			if err := os.WriteFile(path, []byte(tc.body), 0600); err != nil {
				t.Fatal(err)
			}
			rules := guard.ReleaseRules{
				MinimumAge: 48 * time.Hour, StrictMetadata: tc.userStrict,
				OverrideAge: true, ScriptAllowlist: []string{"reviewed@1.0.0"},
			}
			err := applyInstallGuardPolicy(&rules, path)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil || rules.MinimumAge != tc.wantAge || !rules.StrictMetadata ||
				!rules.OverrideAge || strings.Join(rules.ScriptAllowlist, ",") != "reviewed@1.0.0" {
				t.Fatalf("rules = %+v, error = %v", rules, err)
			}
		})
	}
}

func TestPipInputsUseReviewedWheels(t *testing.T) {
	wheels := []guard.PyPIWheel{{
		Filename: "example-1.0.0-py3-none-any.whl",
		URL:      "https://files.pythonhosted.org/packages/example-1.0.0-py3-none-any.whl",
		SHA256:   strings.Repeat("a", 64),
	}}
	links, requirements, err := pipWheelInputs("example==1.0.0", wheels)
	if err != nil || !strings.Contains(string(links), wheels[0].URL) ||
		string(requirements) != "example==1.0.0 --hash=sha256:"+wheels[0].SHA256+"\n" {
		t.Fatalf("pip input: links %q, requirements %q, error %v", links, requirements, err)
	}
	wheels[0].URL = "https://example.invalid/packages/example-1.0.0-py3-none-any.whl"
	if _, _, err := pipWheelInputs("example==1.0.0", wheels); err == nil {
		t.Fatal("accepted a wheel URL outside PyPI's file host")
	}
}

func TestGuardNPMInstallsOnlyReviewedArchive(t *testing.T) {
	ordinary := guardFixtureArchive(t, map[string]string{
		"package/package.json": `{"name":"example","version":"1.0.0"}`,
	})
	scriptManifest := `{"name":"example","version":"1.1.0","scripts":{"install":"node install.cjs"}}`
	scripted := guardFixtureArchive(t, map[string]string{
		"package/package.json": scriptManifest,
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/example":
			fmt.Fprintf(w, `{"name":"example","dist-tags":{"latest":"1.0.0"},"time":{"1.0.0":"2026-11-19T10:00:00Z","1.1.0":"2026-11-01T10:00:00Z"},"versions":{"1.0.0":{"name":"example","version":"1.0.0","dist":{"tarball":%q}},"1.1.0":{"name":"example","version":"1.1.0","scripts":{"install":"node install.cjs"},"dist":{"tarball":%q}}}}`, "http://"+r.Host+"/example.tgz", "http://"+r.Host+"/scripted.tgz")
		case "/example.tgz":
			_, _ = w.Write(ordinary)
		case "/scripted.tgz":
			_, _ = w.Write(scripted)
		default:
			http.NotFound(w, r)
		}
	})
	registry := httptest.NewServer(handler)
	defer registry.Close()

	dir := filepath.Join(t.TempDir(), "new")
	calls := 0
	wantSpec, wantManifest, wantArchive := "example@1.0.0", "", ordinary
	install := func(spec, output, backend string, data []byte, manifest string) error {
		calls++
		if spec != wantSpec || output != dir || backend != "auto" || manifest != wantManifest || !bytes.Equal(data, wantArchive) {
			t.Fatalf("unexpected installer input %q %q %q", spec, output, backend)
		}
		return nil
	}
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	rules := guard.ReleaseRules{}
	ctx := context.Background()
	err := guardNPMInstall(ctx, registry.URL, "example", "", dir, "auto", now, rules, install)
	if err == nil || !strings.Contains(err.Error(), "below the minimum") || calls != 0 {
		t.Fatalf("fresh package reached install: calls %d, error %v", calls, err)
	}
	t.Log(err)
	err = guardNPMInstall(ctx, registry.URL, "example", "1.1.0", dir, "auto", now, rules, install)
	if err == nil || !strings.Contains(err.Error(), "install scripts require") || calls != 0 {
		t.Fatalf("script-bearing package reached install: calls %d, error %v", calls, err)
	}
	err = guardNPMInstall(ctx, registry.URL, "example", "", dir, "auto", now.Add(72*time.Hour), rules, install)
	if err != nil || calls != 1 {
		t.Fatalf("reviewed package was not handed to the installer: calls %d, error %v", calls, err)
	}
	wantSpec, wantManifest, wantArchive = "example@1.1.0", fmt.Sprintf("%x", sha256.Sum256([]byte(scriptManifest))), scripted
	rules.ScriptAllowlist = []string{wantSpec}
	if err := guardNPMInstall(ctx, registry.URL, "example", "1.1.0", dir, "auto", now, rules, install); err != nil || calls != 2 {
		t.Fatalf("allowlisted script release did not reach installer: calls %d, error %v", calls, err)
	}
}

func guardFixtureArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var out bytes.Buffer
	gz := gzip.NewWriter(&out)
	tarball := tar.NewWriter(gz)
	for path, contents := range files {
		if err := tarball.WriteHeader(&tar.Header{Name: path, Mode: 0644, Size: int64(len(contents))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarball.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarball.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func requireGuardSandbox(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("requires a native Linux sandbox")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		if os.Getenv("WARDEN_REQUIRE_AGENT_E2E") == "1" {
			t.Fatal("npm unavailable for the mandatory guard test")
		}
		t.Skip("npm unavailable")
	}
	probe := exec.Command("bwrap", "--ro-bind", "/", "/", "--unshare-net", "--", "/usr/bin/true")
	if result, err := probe.CombinedOutput(); err != nil {
		if os.Getenv("WARDEN_REQUIRE_AGENT_E2E") == "1" {
			t.Fatalf("native sandbox unavailable: %v: %s", err, result)
		}
		t.Skipf("native sandbox unavailable: %v: %s", err, result)
	}
}

func TestGuardApprovedLifecycleRunsOnlyReviewedHooks(t *testing.T) {
	requireGuardSandbox(t)
	manifest := `{"name":"warden-hook-fixture","version":"1.0.0","bundleDependencies":["warden-dep-fixture"],"dependencies":{"warden-dep-fixture":"1.0.0"},"scripts":{"preinstall":"node stamp.cjs pre","install":"node stamp.cjs install","postinstall":"node stamp.cjs post","prepostinstall":"node stamp.cjs extra","prepare":"node stamp.cjs prepare"}}`
	archive := guardFixtureArchive(t, map[string]string{
		"package/package.json": manifest,
		"package/stamp.cjs": `
const fs = require("node:fs");
const stamp = process.argv[2];
if (stamp !== "install") {
  fs.writeFileSync(stamp + ".stamp", "fixture");
} else {
  const proxy = new URL(process.env.HTTP_PROXY);
  require("node:http").get({
    hostname: proxy.hostname, port: proxy.port, path: "http://blocked.warden.invalid/"
  }, (response) => {
    response.resume();
    if (response.statusCode !== 403) throw Error("fixture egress was not denied");
    fs.writeFileSync("install.stamp", "fixture");
  });
}
`,
		"package/node_modules/warden-dep-fixture/package.json": `{"name":"warden-dep-fixture","version":"1.0.0","scripts":{"install":"node stamp.cjs"}}`,
		"package/node_modules/warden-dep-fixture/stamp.cjs":    `require("node:fs").writeFileSync("dep.stamp", "fixture");`,
	})
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(base, "state"))
	dest := filepath.Join(base, "runtime")
	pin := fmt.Sprintf("%x", sha256.Sum256([]byte(manifest)))
	if err := installNPMRuntime("warden-hook-fixture@1.0.0", dest, "linux", archive, pin); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(dest, "node_modules", "warden-hook-fixture")
	for _, name := range []string{"pre", "install", "post"} {
		data, err := os.ReadFile(filepath.Join(pkgDir, name+".stamp"))
		if err != nil || string(data) != "fixture" {
			t.Fatalf("approved hook %q did not run: %q, %v", name, data, err)
		}
	}
	depDir := filepath.Join(pkgDir, "node_modules", "warden-dep-fixture")
	if _, err := os.Stat(filepath.Join(depDir, "package.json")); err != nil {
		t.Fatal("bundled dependency was not installed:", err)
	}
	for _, path := range []string{filepath.Join(pkgDir, "extra.stamp"), filepath.Join(pkgDir, "prepare.stamp"), filepath.Join(depDir, "dep.stamp")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("unrequested hook ran: %v", err)
		}
	}
	auditPath, err := audit.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	log, err := os.Open(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	events, err := audit.ReadEvents(log)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Type == "network" && !event.Allowed && strings.Contains(event.Resource, "blocked.warden.invalid") {
			return
		}
	}
	t.Fatal("approved hook's denied egress was not audited")
}

func TestGuardApprovedBindingGYP(t *testing.T) {
	requireGuardSandbox(t)
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	node, err = filepath.EvalSymlinks(node)
	if err != nil {
		t.Fatal(err)
	}
	headers := filepath.Join(filepath.Dir(filepath.Dir(node)), "include", "node", "node.h")
	if _, err := os.Stat(headers); err != nil {
		t.Skipf("local Node.js headers unavailable: %v", err)
	}
	for _, tool := range []string{"make", "python3"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("native build tool %s unavailable", tool)
		}
	}
	manifest := `{"name":"warden-gyp-fixture","version":"1.0.0"}`
	archive := guardFixtureArchive(t, map[string]string{
		"package/package.json": manifest,
		"package/binding.gyp": `{"targets":[{"target_name":"fixture","type":"none",
			"actions":[{"action_name":"stamp","inputs":[],"outputs":["approved.stamp"],
			"action":["node","-e","require('node:fs').writeFileSync('approved.stamp','fixture')"]}]}]}`,
	})
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(base, "state"))
	dest := filepath.Join(base, "runtime")
	pin := fmt.Sprintf("%x", sha256.Sum256([]byte(manifest)))
	if err := installNPMRuntime("warden-gyp-fixture@1.0.0", dest, "linux", archive, pin); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dest, "node_modules", "warden-gyp-fixture", "approved.stamp"))
	if err != nil || string(data) != "fixture" {
		t.Fatalf("approved binding.gyp did not run: %q, %v", data, err)
	}
}
