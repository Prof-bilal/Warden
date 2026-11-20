package guard

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"
)

// ReviewNPMInstall returns archive bytes only when both the registry and
// archive checks permit installation. A blocked release carries its reasons.
func ReviewNPMInstall(ctx context.Context, client *http.Client, registry, name, version string, now time.Time, rules ReleaseRules) (NPMRelease, []byte, ReleaseDecision, error) {
	release, err := ReadNPMRelease(ctx, client, registry, name, version)
	if err != nil {
		return NPMRelease{}, nil, ReleaseDecision{}, err
	}
	decision := DecideRelease(now, release.Facts, rules)
	if release.Integrity == "" && rules.StrictMetadata {
		decision.Blocks = append(decision.Blocks, "registry archive integrity is unavailable")
	}
	if len(decision.Blocks) > 0 {
		return release, nil, decision, nil
	}

	archive, facts, err := FetchNPMArchive(ctx, client, registry, release)
	if err != nil {
		return release, nil, decision, err
	}
	release.Facts = facts
	decision = DecideRelease(now, facts, rules)
	if release.Integrity == "" {
		decision.Warnings = append(decision.Warnings, "registry archive integrity is unavailable")
	}
	if len(decision.Blocks) > 0 {
		archive = nil
	}
	return release, archive, decision, nil
}

const (
	npmArchiveLimit  = 64 << 20
	npmExpandedLimit = 256 << 20
	npmManifestLimit = 1 << 20
)

// FetchNPMArchive returns the archive bytes together with the inspection
// results so installation can use the inspected archive without a refetch.
func FetchNPMArchive(ctx context.Context, client *http.Client, registry string, release NPMRelease) ([]byte, ReleaseFacts, error) {
	base, err := parseRegistryURL(registry)
	if err != nil {
		return nil, ReleaseFacts{}, err
	}
	if release.Tarball == "" {
		return nil, ReleaseFacts{}, fmt.Errorf("npm metadata did not provide an archive URL")
	}
	archive, err := readRegistry(ctx, client, base, release.Tarball, "application/octet-stream", npmArchiveLimit)
	if err != nil {
		return nil, ReleaseFacts{}, fmt.Errorf("download npm archive for %q at %q: %w", release.Facts.Name, release.Facts.Version, err)
	}
	facts, err := InspectNPMArchive(release, archive)
	if err != nil {
		return nil, ReleaseFacts{}, err
	}
	return archive, facts, nil
}

// InspectNPMArchive checks lifecycle hooks on the same archive bytes the
// installer will receive and merges them with registry release facts.
func InspectNPMArchive(release NPMRelease, archive []byte) (ReleaseFacts, error) {
	if len(archive) > npmArchiveLimit {
		return ReleaseFacts{}, fmt.Errorf("npm archive exceeds the compressed size limit")
	}
	if err := checkNPMIntegrity(release.Integrity, archive); err != nil {
		return ReleaseFacts{}, err
	}
	compressed, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return ReleaseFacts{}, fmt.Errorf("open npm archive: %w", err)
	}
	defer compressed.Close()
	expanded := &io.LimitedReader{R: compressed, N: npmExpandedLimit + 1}
	reader := tar.NewReader(expanded)

	facts := release.Facts
	var manifest *npmVersion
	bindingSeen := false
	for {
		entry, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ReleaseFacts{}, fmt.Errorf("read npm archive: %w", err)
		}
		name := strings.TrimSuffix(entry.Name, "/")
		if path.Clean(name) != name || (name != "package" && !strings.HasPrefix(name, "package/")) ||
			strings.ContainsAny(name, "\\:") {
			return ReleaseFacts{}, fmt.Errorf("unexpected npm archive path %q", entry.Name)
		}
		if entry.Typeflag != tar.TypeReg && entry.Typeflag != tar.TypeDir {
			return ReleaseFacts{}, fmt.Errorf("unexpected npm archive entry type for %q", entry.Name)
		}
		if name == "package" && entry.Typeflag != tar.TypeDir {
			return ReleaseFacts{}, fmt.Errorf("npm archive package root must be a directory")
		}
		for _, component := range strings.Split(name, "/") {
			if strings.TrimRight(component, ". ") != component {
				return ReleaseFacts{}, fmt.Errorf("non-portable npm archive path %q", entry.Name)
			}
		}
		if lower := strings.ToLower(name); lower != name &&
			(lower == "package/package.json" || lower == "package/binding.gyp") {
			return ReleaseFacts{}, fmt.Errorf("unexpected spelling of npm archive entry %q", entry.Name)
		}
		switch name {
		case "package/package.json":
			if manifest != nil || entry.Typeflag != tar.TypeReg || entry.Size > npmManifestLimit {
				return ReleaseFacts{}, fmt.Errorf("npm archive has an invalid or duplicate package.json")
			}
			data, err := io.ReadAll(reader)
			if err != nil {
				return ReleaseFacts{}, fmt.Errorf("read npm archive package.json: %w", err)
			}
			manifest = new(npmVersion)
			if err := json.Unmarshal(data, manifest); err != nil {
				return ReleaseFacts{}, fmt.Errorf("parse npm archive package.json: %w", err)
			}
			facts.NPMManifestSHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
		case "package/binding.gyp":
			if bindingSeen || entry.Typeflag != tar.TypeReg {
				return ReleaseFacts{}, fmt.Errorf("npm archive has an invalid or duplicate binding.gyp")
			}
			bindingSeen = true
		}
	}
	if _, err := io.Copy(tarPadding{}, expanded); err != nil {
		return ReleaseFacts{}, fmt.Errorf("read npm archive footer: %w", err)
	}
	if expanded.N == 0 {
		return ReleaseFacts{}, fmt.Errorf("npm archive exceeds the expanded size limit")
	}
	if manifest == nil || manifest.Name != facts.Name || manifest.Version != facts.Version {
		return ReleaseFacts{}, fmt.Errorf("npm archive does not identify %q at %q", facts.Name, facts.Version)
	}
	facts.InstallScripts = slices.Clone(facts.InstallScripts)
	for _, hook := range npmLifecycleScripts(manifest.Scripts) {
		if !slices.Contains(facts.InstallScripts, hook) {
			facts.InstallScripts = append(facts.InstallScripts, hook)
		}
	}
	facts.HasBindingGYP = facts.HasBindingGYP || manifest.GYPFile || bindingSeen
	return facts, nil
}

// tarPadding accepts zero-filled blocks after the end of the tar stream.
type tarPadding struct{}

func (tarPadding) Write(data []byte) (int, error) {
	if bytes.Count(data, []byte{0}) != len(data) {
		return 0, fmt.Errorf("unexpected data after the tar end marker")
	}
	return len(data), nil
}

func checkNPMIntegrity(integrity string, archive []byte) error {
	if integrity == "" {
		return nil
	}
	// Select the strongest supported algorithm before comparing any hashes.
	items := strings.Fields(integrity)
	for _, algorithm := range []string{"sha512", "sha384", "sha256"} {
		var actual []byte
		for _, item := range items {
			encoded, ok := strings.CutPrefix(item, algorithm+"-")
			if !ok {
				continue
			}
			if actual == nil {
				switch algorithm {
				case "sha512":
					sum := sha512.Sum512(archive)
					actual = sum[:]
				case "sha384":
					sum := sha512.Sum384(archive)
					actual = sum[:]
				case "sha256":
					sum := sha256.Sum256(archive)
					actual = sum[:]
				}
			}
			if expected, err := base64.StdEncoding.DecodeString(encoded); err == nil && bytes.Equal(actual, expected) {
				return nil
			}
		}
		if actual != nil {
			return fmt.Errorf("npm archive does not match registry integrity")
		}
	}
	return fmt.Errorf("npm archive integrity has no supported SHA-2 digest")
}
