package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunOptsAgainstControlledRelease(t *testing.T) {
	const target = "2.0.0-rc.42"
	if fixture := os.Getenv("WARDEN_TEST_UPDATE_FIXTURE"); fixture != "" {
		npmLatestURL = fixture + "/latest"
		apiURL = fixture + "/api/"
		githubDLBase = fixture + "/assets/"
		currentVersion = "0.2.0-beta.5"
		ok, err := RunOpts(os.Stdout, os.Stderr, Options{Yes: true})
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("expected to install the fixture release")
		}
		return
	}

	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("LOCALAPPDATA", filepath.Join(root, "local"))
	name, err := assetName()
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "version.go")
	program := fmt.Sprintf("package main\nimport \"fmt\"\nfunc main() { fmt.Println(%q) }\n", "warden version "+target)
	if err := os.WriteFile(source, []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	asset := filepath.Join(root, name)
	if output, err := exec.Command("go", "build", "-o", asset, source).CombinedOutput(); err != nil {
		t.Fatalf("build fixture: %v\n%s", err, output)
	}
	bytes, err := os.ReadFile(asset)
	if err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x  %s\n", sha256.Sum256(bytes), name)
	startFixture := func(checksum string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/latest":
				fmt.Fprintf(w, `{"version":%q}`, target)
			case "/api/v" + target:
				fmt.Fprint(w, `{"assets":[]}`)
			case "/assets/v" + target + "/" + name:
				_, _ = w.Write(bytes)
			case "/assets/v" + target + "/SHA256SUMS":
				fmt.Fprint(w, checksum)
			default:
				http.NotFound(w, r)
			}
		}))
	}

	// Start the update from a copy within the isolated version cache so the
	// code's self-replacement branch can never target the Go test binary.
	updater, err := InstallPath("0.2.0-beta.5")
	if err != nil {
		t.Fatal(err)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	if err := Replace(updater, original); err != nil {
		t.Fatal(err)
	}
	run := func(url string) ([]byte, error) {
		cmd := exec.Command(updater, "-test.run=^TestRunOptsAgainstControlledRelease$")
		cmd.Env = append(os.Environ(), "WARDEN_TEST_UPDATE_FIXTURE="+url)
		return cmd.CombinedOutput()
	}
	destination, err := InstallPath(target)
	if err != nil {
		t.Fatal(err)
	}
	bad := startFixture(fmt.Sprintf("%s  %s\n", strings.Repeat("0", 64), name))
	output, err := run(bad.URL)
	bad.Close()
	if err == nil || !strings.Contains(string(output), "checksum mismatch") {
		t.Fatalf("bad checksum should fail: %v\n%s", err, output)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("failed update created a cache entry: %v", err)
	}
	good := startFixture(sum)
	output, err = run(good.URL)
	good.Close()
	if err != nil {
		t.Fatalf("verified update: %v\n%s", err, output)
	}
	if err := verifyInstalledBinary(destination, target); err != nil {
		t.Fatal(err)
	}
	if saved, err := os.ReadFile(filepath.Join(filepath.Dir(destination), "SHA256SUMS")); err != nil || string(saved) != sum {
		t.Fatalf("persisted manifest differs: %v", err)
	}
}

func TestParseSums(t *testing.T) {
	in := strings.Join([]string{
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  warden-linux-amd64",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa *warden-darwin-arm64",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb warden-windows-amd64.exe",
		"",
		"not-a-checksum-line",
		"short  name",
	}, "\n")
	got := parseSums([]byte(in))
	if len(got) != 3 {
		t.Fatalf("parseSums got %d entries, want 3: %v", len(got), got)
	}
	if _, ok := got["warden-linux-amd64"]; !ok {
		t.Errorf("warden-linux-amd64 missing: %v", got)
	}
	if _, ok := got["warden-darwin-arm64"]; !ok {
		t.Errorf("binary-mode (*name) entry missing: %v", got)
	}
	if _, ok := got["warden-windows-amd64.exe"]; !ok {
		t.Errorf("warden-windows-amd64.exe missing: %v", got)
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("hello warden")
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])
	name := "warden-test-bin"
	sums := hexSum + "  " + name + "\n"

	if err := verifyChecksum(data, name, sums); err != nil {
		t.Fatalf("verifyChecksum(valid) = %v, want nil", err)
	}
	if err := verifyChecksum([]byte("tampered"), name, sums); err == nil {
		t.Fatal("verifyChecksum(tampered) = nil, want mismatch error")
	} else if !strings.Contains(err.Error(), "mismatch") {
		t.Errorf("unexpected error: %v", err)
	}
	if err := verifyChecksum(data, "other-name", sums); err == nil {
		t.Fatal("verifyChecksum(missing entry) = nil, want error")
	} else if !strings.Contains(err.Error(), "no entry") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.1.10", "0.1.9", 1},
		{"0.1.9", "0.1.10", -1},
		{"0.1.11", "0.1.11", 0},
		{"v0.1.11", "0.1.11", 0},
		{"0.1.11", "v0.1.11", 0},
		{"0.2.0", "0.1.11", 1},
		{"1.0.0", "0.9.9", 1},
		{"0.1.11-rc1", "0.1.11", -1},
		{"0.2.0-beta.2", "0.2.0-beta.10", -1},
		{"0.2.0-beta.1", "0.2.0-alpha.1", 1},
		{"0.2.0", "0.2.0-beta.1", 1},
		{"0.1", "0.1.0", 0}, // missing fields are zero
		{"0.1.0", "0.1", 0},
	}
	for _, tc := range cases {
		got := CompareVersions(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestReportedVersionMatchesExactly(t *testing.T) {
	for _, output := range []string{"warden version 1.2.3", "warden version v1.2.3"} {
		if !reportedVersionMatches(output, "1.2.3") {
			t.Errorf("%q should match", output)
		}
	}
	for _, output := range []string{"warden version 1.2.30", "warden version 11.2.3"} {
		if reportedVersionMatches(output, "1.2.3") {
			t.Errorf("%q must not match", output)
		}
	}
}

func TestReplace(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "warden")
	if err := os.WriteFile(target, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	newBin := []byte("new-binary-contents")
	if err := Replace(target, newBin); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(newBin) {
		t.Fatalf("target content = %q, want %q", got, newBin)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(target)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o755 {
			t.Errorf("mode = %o, want 755", info.Mode().Perm())
		}
	}
	// No temp leftovers.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("expected exactly 1 entry after Replace, got %d", len(entries))
	}
}

func TestCachedChecksumManifestPrivateAndAtomic(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "SHA256SUMS")
	data := []byte(strings.Repeat("a", 64) + "  warden-linux-amd64\n")
	for i := 0; i < 2; i++ {
		if err := replaceFile(target, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != string(data) {
		t.Fatalf("manifest not persisted: %q, %v", got, err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("manifest mode: %o", info.Mode().Perm())
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary manifest files left behind: %v, %v", entries, err)
	}
}

func TestValidateVersion(t *testing.T) {
	ok := []string{"0.1.12", "v0.1.12", "1.0.0", "0.1", "0.2.0-beta.1", "v0.2.0-rc1"}
	for _, in := range ok {
		got, err := validateVersion(in)
		if err != nil {
			t.Errorf("validateVersion(%q) = %v", in, err)
			continue
		}
		if strings.HasPrefix(got, "v") {
			t.Errorf("validateVersion(%q) retained v-prefix: %q", in, got)
		}
	}
	bad := []string{"", "latest", "../x", "0.1.12;rm", "0.1.12/x", "abc", "0.2.0-beta.01", "0.2-beta.1", "0.2.0-beta/1"}
	for _, in := range bad {
		if _, err := validateVersion(in); err == nil {
			t.Errorf("validateVersion(%q) = nil, want error", in)
		}
	}
}

func TestAssetName(t *testing.T) {
	name, err := assetName()
	if err != nil {
		t.Skipf("unsupported platform in test env: %v", err)
	}
	want := "warden-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if name != want {
		t.Errorf("assetName = %q, want %q", name, want)
	}
}
