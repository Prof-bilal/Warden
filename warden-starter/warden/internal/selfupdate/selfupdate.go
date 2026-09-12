// Package selfupdate implements `warden update`: query the npm registry for
// the latest warden-sandbox-cli version, download the matching GitHub Release
// binary, verify it against SHA256SUMS, and install it into the versioned
// cache that the npm wrapper prefers.
//
// Security notes: HTTPS only to pinned hosts (registry.npmjs.org and
// github.com); the binary is installed only after its SHA256 matches the
// release checksum file; replacement is atomic (temp → rename); no shell
// interpolation of version strings; downgrades are refused unless the user
// passes an explicit --version.
package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Overridable for tests.
var (
	npmLatestURL = "https://registry.npmjs.org/warden-sandbox-cli/latest"
	npmPkgURL    = "https://registry.npmjs.org/warden-sandbox-cli/"
	apiURL       = "https://api.github.com/repos/Prof-bilal/Warden/releases/tags/"
	githubDLBase = "https://github.com/Prof-bilal/Warden/releases/download/"
)

const (
	apiTimeout  = 15 * time.Second
	dlTimeout   = 120 * time.Second
	maxBodySize = 64 << 20 // 64 MiB cap on binary downloads
	npmPackage  = "warden-sandbox-cli"
)

// currentVersion is set by main to version.Version; kept as a var so tests
// can stub it.
var currentVersion = "dev"

// Options controls a single update invocation.
type Options struct {
	// CheckOnly reports current vs latest without downloading.
	CheckOnly bool
	// TargetVersion, when non-empty, installs that exact version instead of latest.
	TargetVersion string
	// Yes skips the interactive confirmation prompt (required for non-TTY).
	Yes bool
}

// userAgent identifies the updater to GitHub/npm (both require a UA header).
func userAgent() string {
	return "warden-selfupdate/" + currentVersion
}

type npmLatest struct {
	Version string `json:"version"`
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

// validateVersion rejects anything that is not a strict dotted numeric semver
// (optional leading v). Never interpolate untrusted version strings into paths
// or URLs without this gate.
func validateVersion(v string) (string, error) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return "", fmt.Errorf("version must not be empty")
	}
	if strings.ContainsAny(v, "/\\|*?;<>$`\"'\n\r\t ") {
		return "", fmt.Errorf("version %q contains illegal characters", v)
	}
	parts := strings.Split(v, ".")
	if len(parts) < 1 || len(parts) > 4 {
		return "", fmt.Errorf("version %q is not a dotted numeric semver", v)
	}
	for _, p := range parts {
		if p == "" {
			return "", fmt.Errorf("version %q has an empty component", v)
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				// Allow a single prerelease suffix only after stripping above;
				// we require pure numeric components.
				return "", fmt.Errorf("version %q must be numeric dotted (got component %q)", v, p)
			}
		}
	}
	return v, nil
}

func getJSON(url string, dest any) error {
	client := &http.Client{Timeout: apiTimeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent())
	req.Header.Set("Accept", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d fetching %s", res.StatusCode, url)
	}
	return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(dest)
}

// FetchLatestNPM returns the latest version published to the npm registry.
func FetchLatestNPM() (string, error) {
	var meta npmLatest
	if err := getJSON(npmLatestURL, &meta); err != nil {
		return "", fmt.Errorf("cannot reach npm registry: %w", err)
	}
	return validateVersion(meta.Version)
}

// npmVersionExists confirms a specific version is published on npm.
func npmVersionExists(version string) error {
	var meta npmLatest
	url := npmPkgURL + version
	if err := getJSON(url, &meta); err != nil {
		return fmt.Errorf("npm has no warden-sandbox-cli@%s: %w", version, err)
	}
	got, err := validateVersion(meta.Version)
	if err != nil {
		return err
	}
	if got != version {
		return fmt.Errorf("npm returned version %s for request %s", got, version)
	}
	return nil
}

// fetchRelease returns download URLs for the GitHub release assets for
// v<version>. Prefers the Releases API; falls back to constructing the
// canonical download URLs (checksum verification still protects integrity).
func fetchRelease(version string) (binURL, sumsURL string, err error) {
	name, err := assetName()
	if err != nil {
		return "", "", err
	}
	var rel release
	if err := getJSON(apiURL+"v"+version, &rel); err == nil {
		for _, a := range rel.Assets {
			switch a.Name {
			case name:
				binURL = a.BrowserDownloadURL
			case "SHA256SUMS":
				sumsURL = a.BrowserDownloadURL
			}
		}
	}
	if binURL == "" {
		binURL = githubDLBase + "v" + version + "/" + name
	}
	if sumsURL == "" {
		sumsURL = githubDLBase + "v" + version + "/SHA256SUMS"
	}
	return binURL, sumsURL, nil
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

// cacheDir is ~/.cache/warden (or %LOCALAPPDATA%\warden on Windows).
func cacheDir() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "warden"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "warden"), nil
}

// InstallPath returns the versioned cache path the npm wrapper will prefer.
func InstallPath(version string) (string, error) {
	name, err := assetName()
	if err != nil {
		return "", err
	}
	base, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, version, name), nil
}

// Replace atomically installs data at target. On Windows the running
// executable cannot be overwritten in place, so it is renamed aside first.
// Exported for tests.
func Replace(target string, data []byte) error {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating install dir: %w", err)
	}
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
	if runtime.GOOS == "windows" {
		old := target + ".old"
		_ = os.Remove(old)
		if err := os.Rename(target, old); err != nil && !os.IsNotExist(err) {
			os.Remove(tmpName)
			return fmt.Errorf("moving current binary aside: %w", err)
		}
		if err := os.Rename(tmpName, target); err != nil {
			_ = os.Rename(old, target)
			return fmt.Errorf("installing new binary: %w", err)
		}
		_ = os.Remove(old)
		return nil
	}
	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("installing new binary: %w", err)
	}
	return nil
}

// Check returns the running version and the latest published npm version.
func Check() (current, latest string, err error) {
	latest, err = FetchLatestNPM()
	if err != nil {
		return currentVersion, "", err
	}
	return currentVersion, latest, nil
}

// SetCurrentVersion overrides the version used for update comparisons.
// main calls it with the ldflags-stamped version so `warden update` never
// compares against the in-source "dev" default. Exported for main/tests.
func SetCurrentVersion(v string) {
	if v != "" {
		currentVersion = strings.TrimPrefix(strings.TrimSpace(v), "v")
	}
}

// verifyInstalledBinary runs the installed binary with --version and checks
// that the reported version matches the target.
func verifyInstalledBinary(binPath, wantVersion string) error {
	cmd := exec.Command(binPath, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("post-install verification failed (%v): %s", err, strings.TrimSpace(string(out)))
	}
	got := strings.TrimSpace(string(out))
	want := strings.TrimPrefix(strings.TrimSpace(wantVersion), "v")
	// The version command may include a product prefix, but the version itself
	// must be a complete token. Substring matching accepts 1.2.30 for 1.2.3.
	if !reportedVersionMatches(got, want) {
		return fmt.Errorf("post-install verification: binary reported %q, want version %q", got, want)
	}
	return nil
}

func reportedVersionMatches(output, want string) bool {
	want = strings.TrimPrefix(strings.TrimSpace(want), "v")
	for _, field := range strings.Fields(output) {
		if strings.TrimPrefix(field, "v") == want {
			return true
		}
	}
	return false
}

// Run performs the full update flow, writing progress to stderr and the
// final verdict to stdout. updated=false with nil error means already
// current. Exported for tests.
func Run(stdout, stderr io.Writer) (bool, error) {
	return RunOpts(stdout, stderr, Options{})
}

// RunOpts is Run with explicit options (--check, --version, --yes).
func RunOpts(stdout, stderr io.Writer, opts Options) (bool, error) {
	name, err := assetName()
	if err != nil {
		return false, err
	}

	fmt.Fprintf(stderr, "Checking npm registry for %s updates...\n", npmPackage)
	latest, err := FetchLatestNPM()
	if err != nil {
		return false, err
	}

	target := latest
	explicit := false
	if opts.TargetVersion != "" {
		target, err = validateVersion(opts.TargetVersion)
		if err != nil {
			return false, err
		}
		if err := npmVersionExists(target); err != nil {
			return false, err
		}
		explicit = true
	}

	fmt.Fprintf(stderr, "Current: %s\n", currentVersion)
	fmt.Fprintf(stderr, "Latest:  %s\n", latest)
	if explicit {
		fmt.Fprintf(stderr, "Target:  %s\n", target)
	}

	if opts.CheckOnly {
		if currentVersion != "dev" && CompareVersions(currentVersion, latest) >= 0 {
			fmt.Fprintf(stdout, "warden version %s -- already up to date\n", currentVersion)
			return false, nil
		}
		fmt.Fprintf(stdout, "Update available: %s → %s\n", currentVersion, latest)
		fmt.Fprintf(stderr, "Run `warden update` to install.\n")
		return false, nil
	}

	if currentVersion != "dev" && !explicit && CompareVersions(currentVersion, target) >= 0 {
		fmt.Fprintf(stdout, "warden version %s -- already up to date\n", currentVersion)
		return false, nil
	}
	if currentVersion != "dev" && explicit && CompareVersions(currentVersion, target) == 0 {
		fmt.Fprintf(stdout, "warden version %s -- already at requested version\n", currentVersion)
		return false, nil
	}
	if currentVersion != "dev" && explicit && CompareVersions(target, currentVersion) < 0 {
		return false, fmt.Errorf("refusing to downgrade from %s to %s (pass a higher --version)", currentVersion, target)
	}
	if currentVersion == "dev" {
		fmt.Fprintf(stderr, "Current build is unstamped (\"dev\"); installing v%s.\n", target)
	}

	fmt.Fprintf(stderr, "Will install: warden %s (%s)\n", target, name)
	fmt.Fprintf(stderr, "Source: GitHub Release v%s (SHA256 verified)\n", target)
	fmt.Fprintf(stderr, "Destination: versioned cache (~/.cache/warden/%s/)\n", target)

	if !opts.Yes {
		fi, _ := os.Stderr.Stat()
		isTTY := fi != nil && fi.Mode()&os.ModeCharDevice != 0
		if !isTTY || os.Getenv("CI") != "" {
			return false, fmt.Errorf("non-interactive update requires --yes (refusing to modify the install without confirmation)")
		}
		fmt.Fprintf(stderr, "Proceed? [y/N] ")
		var answer string
		_, _ = fmt.Fscanln(os.Stdin, &answer)
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintf(stderr, "Update cancelled.\n")
			return false, nil
		}
	}

	binURL, sumsURL, err := fetchRelease(target)
	if err != nil {
		return false, err
	}

	fmt.Fprintf(stderr, "Downloading warden v%s (%s)...\n", target, name)
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

	dest, err := InstallPath(target)
	if err != nil {
		return false, err
	}
	fmt.Fprintf(stderr, "Installing v%s (verified) -> %s\n", target, dest)
	if err := Replace(dest, bin); err != nil {
		return false, err
	}
	if err := verifyInstalledBinary(dest, target); err != nil {
		_ = os.Remove(dest)
		return false, err
	}

	// Also replace the running executable when it is writable and not itself
	// inside the versioned cache (e.g. a manual /usr/local/bin install).
	if self, err := os.Executable(); err == nil {
		if self, err = filepath.EvalSymlinks(self); err == nil {
			if !isWardenCachePath(self) {
				if err := Replace(self, bin); err != nil {
					fmt.Fprintf(stderr, "note: cache install succeeded; could not replace running binary %s: %v\n", self, err)
				}
			}
		}
	}

	fmt.Fprintf(stdout, "Updated to warden version %s\n", target)
	fmt.Fprintf(stderr, "Note: if you installed via npm, also run:\n  npm install -g %s@%s\nto update the launcher package metadata.\n", npmPackage, target)
	return true, nil
}

func isWardenCachePath(path string) bool {
	base, err := cacheDir()
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(base, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
