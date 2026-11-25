package guard

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

const DefaultReleaseAge = 48 * time.Hour

// ReleaseFacts describes an exact registry release. Nil maintainer slices
// indicate that the registry lacked one side of the comparison.
type ReleaseFacts struct {
	Name                 string
	Version              string
	PublishedAt          time.Time
	InstallScripts       []string
	HasBindingGYP        bool
	NPMManifestSHA256    string
	NPMCanInstallOffline bool
	Maintainers          []string
	PreviousMaintainers  []string
}

type ReleaseRules struct {
	MinimumAge      time.Duration
	StrictMetadata  bool
	OverrideAge     bool
	ScriptAllowlist []string
}

type ReleaseDecision struct {
	Blocks   []string
	Warnings []string
}

// DecideRelease evaluates registry and manifest facts before an install.
// The caller must apply the decision to the same exact release in the sandbox.
func DecideRelease(now time.Time, facts ReleaseFacts, rules ReleaseRules) ReleaseDecision {
	var result ReleaseDecision
	if !npmReleaseName.MatchString(facts.Name) || !validReleaseVersion(facts.Version) {
		result.Blocks = append(result.Blocks, "an exact package name and version are required")
		return result
	}
	if rules.MinimumAge == 0 {
		rules.MinimumAge = DefaultReleaseAge
	}
	if rules.MinimumAge < 0 {
		result.Blocks = append(result.Blocks, "minimum release age must be positive")
		return result
	}

	switch age := now.Sub(facts.PublishedAt); {
	case facts.PublishedAt.IsZero():
		if rules.StrictMetadata {
			result.Blocks = append(result.Blocks, "registry publication time is unavailable")
		} else {
			result.Warnings = append(result.Warnings, "registry publication time is unavailable")
		}
	case age < 0:
		result.Blocks = append(result.Blocks, "registry publication time is in the future")
	case age < rules.MinimumAge:
		message := fmt.Sprintf("release age %s is below the minimum %s", age.Round(time.Second), rules.MinimumAge)
		if rules.OverrideAge {
			result.Warnings = append(result.Warnings, message+" (override requested)")
		} else {
			result.Blocks = append(result.Blocks, message)
		}
	}

	scripts := facts.InstallScripts
	if facts.HasBindingGYP {
		scripts = append(slices.Clone(scripts), "binding.gyp")
	}
	if len(scripts) > 0 && !slices.Contains(rules.ScriptAllowlist, facts.Name+"@"+facts.Version) {
		result.Blocks = append(result.Blocks, fmt.Sprintf("install scripts require an exact allowlist entry: %q", scripts))
	}

	if facts.Maintainers != nil && facts.PreviousMaintainers != nil {
		current := slices.Compact(slices.Sorted(slices.Values(facts.Maintainers)))
		previous := slices.Compact(slices.Sorted(slices.Values(facts.PreviousMaintainers)))
		if !slices.Equal(current, previous) {
			result.Warnings = append(result.Warnings, "registry maintainers changed")
		}
	}
	return result
}

func validReleaseVersion(version string) bool {
	const versionChars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ.-_+!"
	return version != "" && version[0] >= '0' && version[0] <= '9' && strings.Trim(version, versionChars) == ""
}
