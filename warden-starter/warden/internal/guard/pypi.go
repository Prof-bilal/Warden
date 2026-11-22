package guard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const pypiJSONLimit = 16 << 20

var (
	pythonDistributionName = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?$`)
	pythonNameSeparator    = regexp.MustCompile(`[-_.]+`)
)

type PyPIRelease struct {
	Name                string
	Version             string
	Wheels              []PyPIWheel
	SourceDistributions int
}

type PyPIWheel struct {
	Filename   string
	URL        string
	SHA256     string
	UploadedAt time.Time
}

// ReadPyPIRelease reads the latest release or an exact version from PyPI's
// JSON API. Wheel compatibility is left to the installer.
func ReadPyPIRelease(ctx context.Context, client *http.Client, registry, name, version string) (PyPIRelease, error) {
	if !pythonDistributionName.MatchString(name) || (version != "" && !validReleaseVersion(version)) {
		return PyPIRelease{}, fmt.Errorf("invalid Python distribution name or version")
	}
	base, err := parseRegistryURL(registry)
	if err != nil {
		return PyPIRelease{}, err
	}
	name = normalizePythonName(name)
	endpoint := strings.TrimRight(base.String(), "/") + "/pypi/" + url.PathEscape(name)
	if version != "" {
		endpoint += "/" + url.PathEscape(version)
	}
	raw, err := readRegistry(ctx, client, base, endpoint+"/json", "application/json", pypiJSONLimit)
	if err != nil {
		return PyPIRelease{}, fmt.Errorf("fetch PyPI metadata for %q: %w", name, err)
	}
	return parsePyPIRelease(raw, name, version)
}

func parsePyPIRelease(raw []byte, name, version string) (PyPIRelease, error) {
	var document struct {
		Info struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"info"`
		Files []struct {
			Filename   string `json:"filename"`
			URL        string `json:"url"`
			Type       string `json:"packagetype"`
			UploadedAt string `json:"upload_time_iso_8601"`
			Yanked     bool   `json:"yanked"`
			Digests    struct {
				SHA256 string `json:"sha256"`
			} `json:"digests"`
		} `json:"urls"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		return PyPIRelease{}, fmt.Errorf("parse PyPI metadata for %q: %w", name, err)
	}
	info := document.Info
	if !pythonDistributionName.MatchString(info.Name) || normalizePythonName(info.Name) != name ||
		!validReleaseVersion(info.Version) || (version != "" && version != info.Version) {
		return PyPIRelease{}, fmt.Errorf("PyPI metadata has no matching release for %q at %q", name, version)
	}

	release := PyPIRelease{Name: name, Version: info.Version}
	for _, file := range document.Files {
		if file.Type == "sdist" {
			release.SourceDistributions++
		}
		if file.Yanked || file.Type != "bdist_wheel" || !strings.HasSuffix(file.Filename, ".whl") ||
			strings.ContainsAny(file.Filename, "/\\: \t\r\n") {
			continue
		}
		uploadedAt, _ := time.Parse(time.RFC3339Nano, file.UploadedAt)
		release.Wheels = append(release.Wheels, PyPIWheel{
			Filename: file.Filename, URL: file.URL, SHA256: file.Digests.SHA256, UploadedAt: uploadedAt,
		})
	}
	return release, nil
}

func normalizePythonName(name string) string {
	return pythonNameSeparator.ReplaceAllString(strings.ToLower(name), "-")
}

// ReviewPyPIWheels applies the cooldown to each individual upload. The
// installer must use only the returned wheel hashes and must stay binary-only.
func ReviewPyPIWheels(release PyPIRelease, now time.Time, rules ReleaseRules) ([]PyPIWheel, ReleaseDecision) {
	var decision ReleaseDecision
	if release.SourceDistributions > 0 {
		decision.Warnings = append(decision.Warnings, "source distributions are excluded from binary-only installs")
	}
	var eligible []PyPIWheel
	for _, wheel := range release.Wheels {
		hash, err := hex.DecodeString(wheel.SHA256)
		if err != nil || len(hash) != sha256.Size || wheel.URL == "" {
			decision.Warnings = append(decision.Warnings, fmt.Sprintf("skipping wheel %q without a URL and SHA-256", wheel.Filename))
			continue
		}
		check := DecideRelease(now, ReleaseFacts{Name: release.Name, Version: release.Version, PublishedAt: wheel.UploadedAt}, rules)
		for _, warning := range check.Warnings {
			decision.Warnings = append(decision.Warnings, fmt.Sprintf("wheel %q: %s", wheel.Filename, warning))
		}
		if len(check.Blocks) != 0 {
			decision.Warnings = append(decision.Warnings, fmt.Sprintf("skipping wheel %q: %s", wheel.Filename, strings.Join(check.Blocks, ", ")))
			continue
		}
		eligible = append(eligible, wheel)
	}
	if len(eligible) == 0 {
		decision.Blocks = append(decision.Blocks, "no eligible binary wheel for the reviewed release")
	}
	return eligible, decision
}
