package guard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const npmPackumentLimit = 16 << 20

var npmReleaseName = regexp.MustCompile(`^(@[A-Za-z0-9-][A-Za-z0-9._-]*/)?[A-Za-z0-9-][A-Za-z0-9._-]*$`)

type NPMRelease struct {
	Facts     ReleaseFacts
	Tarball   string
	Integrity string
}

type npmMaintainer struct {
	Name string `json:"name"`
}

type npmVersion struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Scripts          map[string]string `json:"scripts"`
	HasInstallScript bool              `json:"hasInstallScript"`
	GYPFile          bool              `json:"gypfile"`
	Maintainers      []npmMaintainer   `json:"maintainers"`
	Dist             struct {
		Tarball   string `json:"tarball"`
		Integrity string `json:"integrity"`
	} `json:"dist"`
}

// ReadNPMRelease resolves an exact version or the latest tag against a full
// registry document. The caller verifies the archive and uses the resolved
// version for installation.
func ReadNPMRelease(ctx context.Context, client *http.Client, registry, name, version string) (NPMRelease, error) {
	if !npmReleaseName.MatchString(name) {
		return NPMRelease{}, fmt.Errorf("invalid npm package name %q", name)
	}
	base, err := parseRegistryURL(registry)
	if err != nil {
		return NPMRelease{}, err
	}
	endpoint := strings.TrimRight(base.String(), "/") + "/" + url.PathEscape(name)
	raw, err := readRegistry(ctx, client, base, endpoint, "application/json", npmPackumentLimit)
	if err != nil {
		return NPMRelease{}, fmt.Errorf("fetch npm metadata for %q: %w", name, err)
	}
	return parseNPMRelease(raw, name, version)
}

func parseRegistryURL(registry string) (*url.URL, error) {
	base, err := url.Parse(registry)
	if err != nil || base.Hostname() == "" || base.User != nil || base.ForceQuery || base.RawQuery != "" ||
		base.Fragment != "" || (base.Scheme != "https" && base.Scheme != "http") {
		return nil, fmt.Errorf("invalid registry URL")
	}
	return base, nil
}

func readRegistry(ctx context.Context, client *http.Client, base *url.URL, endpoint, accept string, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if req.URL.Scheme != base.Scheme || req.URL.Host != base.Host || req.URL.User != nil {
		return nil, fmt.Errorf("download URL leaves the configured registry")
	}
	req.Header.Set("Accept", accept)
	if accept == "application/octet-stream" {
		req.Header.Set("Accept-Encoding", "identity")
	}
	if client == nil {
		client = http.DefaultClient
	}
	registryClient := *client
	redirect := registryClient.CheckRedirect
	registryClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 || req.URL.Scheme != base.Scheme || req.URL.Host != base.Host || req.URL.User != nil {
			return fmt.Errorf("registry redirect leaves the configured registry or exceeds the limit")
		}
		if redirect != nil {
			return redirect(req, via)
		}
		return nil
	}
	resp, err := registryClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > limit {
		return nil, fmt.Errorf("registry response exceeds the size limit")
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("registry response exceeds the size limit")
	}
	return raw, nil
}

func parseNPMRelease(raw []byte, name, version string) (NPMRelease, error) {
	var document struct {
		Name     string                `json:"name"`
		DistTags map[string]string     `json:"dist-tags"`
		Versions map[string]npmVersion `json:"versions"`
		Time     map[string]string     `json:"time"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		return NPMRelease{}, fmt.Errorf("parse npm metadata for %q: %w", name, err)
	}
	if version == "" || version == "latest" {
		version = document.DistTags["latest"]
	}
	manifest, found := document.Versions[version]
	if !found || version == "" || document.Name != name || manifest.Name != name || manifest.Version != version {
		return NPMRelease{}, fmt.Errorf("npm metadata has no matching release for %q at %q", name, version)
	}

	publishedAt, _ := time.Parse(time.RFC3339Nano, document.Time[version])
	facts := ReleaseFacts{
		Name:          name,
		Version:       version,
		PublishedAt:   publishedAt,
		HasBindingGYP: manifest.GYPFile,
		Maintainers:   npmMaintainerNames(manifest.Maintainers),
	}
	facts.InstallScripts = npmLifecycleScripts(manifest.Scripts)
	if manifest.HasInstallScript && len(facts.InstallScripts) == 0 {
		facts.InstallScripts = []string{"registry reports install script"}
	}

	var previousAt time.Time
	var previousVersion string
	for version, candidate := range document.Versions {
		if candidate.Name != name || candidate.Version != version {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, document.Time[version])
		if err == nil && at.Before(publishedAt) && (at.After(previousAt) || (at.Equal(previousAt) && version > previousVersion)) {
			previousAt, previousVersion = at, version
			facts.PreviousMaintainers = npmMaintainerNames(candidate.Maintainers)
		}
	}
	return NPMRelease{Facts: facts, Tarball: manifest.Dist.Tarball, Integrity: manifest.Dist.Integrity}, nil
}

func npmLifecycleScripts(scripts map[string]string) []string {
	var hooks []string
	for _, hook := range []string{"preinstall", "install", "postinstall", "prepare"} {
		if scripts[hook] != "" {
			hooks = append(hooks, hook)
		}
	}
	return hooks
}

func npmMaintainerNames(maintainers []npmMaintainer) []string {
	if maintainers == nil {
		return nil
	}
	names := make([]string, 0, len(maintainers))
	for _, maintainer := range maintainers {
		if maintainer.Name != "" {
			names = append(names, maintainer.Name)
		}
	}
	return names
}
