package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

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
		{"0.1.11-rc1", "0.1.11", 0}, // suffix ignored
		{"0.1", "0.1.0", 0},         // missing fields are zero
		{"0.1.0", "0.1", 0},
	}
	for _, tc := range cases {
		got := CompareVersions(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
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

func TestValidateVersion(t *testing.T) {
	ok := []string{"0.1.12", "v0.1.12", "1.0.0", "0.1"}
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
	bad := []string{"", "latest", "../x", "0.1.12;rm", "0.1.12/x", "abc", "0.1.12-rc1"}
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
