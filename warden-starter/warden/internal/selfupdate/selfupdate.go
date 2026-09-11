// Package selfupdate implements `warden update`: fetch the latest GitHub
// Release, download the platform binary, verify it against the published
// SHA256SUMS, and atomically replace the running executable.
//
// Security notes: the downloaded binary is only trusted after its SHA256
// matches the checksum file from the same release; the replacement is a
// same-directory rename so an interrupted update never leaves a broken
// install. HTTPS only, no proxy involvement, no arbitrary URLs.
package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Overridable for tests.
var apiURL = "https://api.github.com/repos/Prof-bilal/Warden/releases/latest"

const (
	apiTimeout  = 15 * time.Second
	dlTimeout   = 120 * time.Second
	maxBodySize = 64 << 20 // 64 MiB cap on binary downloads
)

// currentVersion is set by main to version.Version; kept as a var so tests
// can stub it.
var currentVersion = "dev"

// userAgent identifies the updater to GitHub (the API requires a UA header).
func userAgent() string {
	return "warden-selfupdate/" + currentVersion
}

type release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// assetName maps GOOS/GOARCH to the release asset produced by the Makefile.
func assetName() (string, error) {
	osMap := map[string]string{"linux": "linux", "darwin": "darwin", "windows": "windows"}
	archMap := map[string]string{"amd64": "amd64", "arm64": "arm64"}
	o, ok := osMap[runtime.GOOS]
	if !ok {
		return "", fmt.Errorf("unsupported GOOS %q", runtime.GOOS)
	}
	a, ok := archMap[runtime.GOARCH]
	if !ok {
		return "", fmt.Errorf("unsupported GOARCH %q", runtime.GOARCH)
	}
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("warden-%s-%s%s", o, a, ext), nil
}

// fetchLatest returns the latest release tag and its assets.
func fetchLatest() (*release, error) {
	client := &http.Client{Timeout: apiTimeout}
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent())
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release lookup failed: HTTP %d", res.StatusCode)
	}
	var rel release
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&rel); err != nil {
		return nil, fmt.Errorf("parsing release metadata: %w", err)
	}
	if rel.TagName == "" || len(rel.Assets) == 0 {
		return nil, fmt.Errorf("release lookup returned no assets")
	}
	return &rel, nil
}

// getBody downloads a release asset with a hard size cap.
func getBody(url string) ([]byte, error) {
	client := &http.Client{Timeout: dlTimeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent())
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: HTTP %d for %s", res.StatusCode, url)
	}
	return io.ReadAll(io.LimitReader(res.Body, maxBodySize))
}

// parseSums parses a sha256sum-style file into a name-to-hex digest map.
// Exported for tests.
func parseSums(data []byte) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "<hex>  <name>" (two spaces); tolerate a single space and
		// the " *name" binary-mode marker produced by some tools.
		parts := strings.Fields(line)
		if len(parts) < 2 || len(parts[0]) != 64 {
			continue
		}
		if _, err := hex.DecodeString(parts[0]); err != nil {
			continue
		}
		name := strings.TrimPrefix(line, parts[0])
		name = strings.TrimLeft(name, " ")
		name = strings.TrimPrefix(name, "*")
		if name == "" {
			continue
		}
		out[name] = parts[0]
	}
	return out
}

// verifyChecksum confirms data matches the published digest for name.
// Exported for tests.
func verifyChecksum(data []byte, name, sumsData string) error {
	sums := parseSums([]byte(sumsData))
	want, ok := sums[name]
	if !ok {
		return fmt.Errorf("checksum file has no entry for %s -- refusing to install", name)
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != want {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", name, got, want)
	}
	return nil
}

// CompareVersions returns -1, 0, or 1 comparing dotted numeric versions
// (v-prefix optional; prerelease/build suffixes are ignored).
// Exported for tests.
func CompareVersions(a, b string) int {
	norm := func(s string) []int {
		s = strings.TrimPrefix(strings.TrimSpace(s), "v")
		if i := strings.IndexAny(s, "-+"); i >= 0 {
			s = s[:i]
		}
		fields := strings.Split(s, ".")
		out := make([]int, 0, len(fields))
		for _, f := range fields {
			var n int
			fmt.Sscanf(f, "%d", &n)
			out = append(out, n)
		}
		return out
	}
	av, bv := norm(a), norm(b)
	for i := 0; i < len(av) || i < len(bv); i++ {
		var x, y int
		if i < len(av) {
			x = av[i]
		}
		if i < len(bv) {
			y = bv[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

// Replace atomically installs data at target. On Windows the running
// executable cannot be overwritten in place, so it is renamed aside first.
// Exported for tests.
func Replace(target string, data []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".warden-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("writing new binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("closing new binary: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpName, 0o755); err != nil {
			os.Remove(tmpName)
			return fmt.Errorf("setting mode: %w", err)
		}
	}
	// Same-directory rename: atomic on POSIX. On Windows a running exe
	// cannot be truncated but can be renamed out of the way.
	if runtime.GOOS == "windows" {
		old := target + ".old"
		_ = os.Remove(old) // best effort: leftover from a previous update
		if err := os.Rename(target, old); err != nil && !os.IsNotExist(err) {
			os.Remove(tmpName)
			return fmt.Errorf("moving current binary aside: %w", err)
		}
		if err := os.Rename(tmpName, target); err != nil {
			_ = os.Rename(old, target) // roll back
			return fmt.Errorf("installing new binary: %w", err)
		}
		_ = os.Remove(old) // may fail while the old process still runs; harmless
		return nil
	}
	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("installing new binary: %w", err)
	}
	return nil
}

// Check returns the running version and the latest published version.
func Check() (current, latest string, err error) {
	rel, err := fetchLatest()
	if err != nil {
		return currentVersion, "", err
	}
	return currentVersion, strings.TrimPrefix(rel.TagName, "v"), nil
}

// SetCurrentVersion overrides the version used for update comparisons.
// main calls it with the ldflags-stamped version so `warden update` never
// compares against the in-source "dev" default. Exported for main/tests.
func SetCurrentVersion(v string) {
	if v != "" {
		currentVersion = v
	}
}

// Run performs the full update flow, writing progress to stderr and the
// final verdict to stdout. updated=false with nil error means already
// current. Exported for tests.
func Run(stdout, stderr io.Writer) (bool, error) {
	name, err := assetName()
	if err != nil {
		return false, err
	}
	fmt.Fprintf(stderr, "Checking for updates...\n")
	rel, err := fetchLatest()
	if err != nil {
		return false, fmt.Errorf("cannot reach GitHub Releases: %w", err)
	}
	latest := strings.TrimPrefix(rel.TagName, "v")
	if currentVersion != "dev" && CompareVersions(currentVersion, latest) >= 0 {
		fmt.Fprintf(stdout, "warden version %s -- already up to date\n", currentVersion)
		return false, nil
	}
	if currentVersion == "dev" {
		fmt.Fprintf(stderr, "Current build is unstamped (\"dev\"); installing latest release v%s.\n", latest)
	}

	var binURL, sumsURL string
	for _, a := range rel.Assets {
		switch a.Name {
		case name:
			binURL = a.BrowserDownloadURL
		case "SHA256SUMS":
			sumsURL = a.BrowserDownloadURL
		}
	}
	if binURL == "" {
		return false, fmt.Errorf("release v%s has no asset %s", latest, name)
	}
	if sumsURL == "" {
		return false, fmt.Errorf("release v%s has no SHA256SUMS -- refusing to install an unverified binary", latest)
	}

	fmt.Fprintf(stderr, "Downloading warden v%s (%s)...\n", latest, name)
	bin, err := getBody(binURL)
	if err != nil {
		return false, err
	}
	sums, err := getBody(sumsURL)
	if err != nil {
		return false, fmt.Errorf("downloading checksums: %w", err)
	}
	if err := verifyChecksum(bin, name, string(sums)); err != nil {
		return false, err
	}

	self, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("locating running binary: %w", err)
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		return false, fmt.Errorf("resolving running binary: %w", err)
	}
	fmt.Fprintf(stderr, "Installing v%s (verified) -> %s\n", latest, self)
	if err := Replace(self, bin); err != nil {
		return false, err
	}
	fmt.Fprintf(stdout, "Updated to warden version %s\n", latest)
	return true, nil
}
