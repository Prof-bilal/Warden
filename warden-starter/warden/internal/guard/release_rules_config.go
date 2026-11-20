package guard

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const guardConfigLimit = 1 << 20

// ReadReleaseRules loads guard.yaml. A missing file uses the default rules.
// An age override is per invocation and is deliberately absent from the file.
func ReadReleaseRules(path string) (ReleaseRules, error) {
	rules := ReleaseRules{MinimumAge: DefaultReleaseAge}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return rules, nil
	}
	if err != nil {
		return ReleaseRules{}, fmt.Errorf("inspect guard config %q: %w", path, err)
	}
	if !info.Mode().IsRegular() || info.Size() > guardConfigLimit {
		return ReleaseRules{}, fmt.Errorf("guard config %q must be a regular file no larger than %d bytes", path, guardConfigLimit)
	}
	file, err := os.Open(path)
	if err != nil {
		return ReleaseRules{}, fmt.Errorf("open guard config %q: %w", path, err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, guardConfigLimit+1))
	if err != nil {
		return ReleaseRules{}, fmt.Errorf("read guard config %q: %w", path, err)
	}
	if len(data) > guardConfigLimit {
		return ReleaseRules{}, fmt.Errorf("guard config %q exceeds the size limit", path)
	}

	var settings struct {
		MinimumReleaseAge string   `yaml:"minimum_release_age"`
		Strict            bool     `yaml:"strict"`
		ScriptAllowlist   []string `yaml:"script_allowlist"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&settings); err != nil {
		return ReleaseRules{}, fmt.Errorf("parse guard config %q: %w", path, err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return ReleaseRules{}, fmt.Errorf("guard config %q must contain one YAML document", path)
	}
	if settings.MinimumReleaseAge != "" {
		age, err := time.ParseDuration(settings.MinimumReleaseAge)
		if err != nil || age <= 0 {
			return ReleaseRules{}, fmt.Errorf("minimum_release_age must be a positive duration, such as 48h")
		}
		rules.MinimumAge = age
	}
	for _, item := range settings.ScriptAllowlist {
		split := strings.LastIndexByte(item, '@')
		if split <= 0 || !npmReleaseName.MatchString(item[:split]) || !validReleaseVersion(item[split+1:]) {
			return ReleaseRules{}, fmt.Errorf("script_allowlist entry %q must be an exact name@version", item)
		}
	}
	rules.StrictMetadata = settings.Strict
	rules.ScriptAllowlist = settings.ScriptAllowlist
	return rules, nil
}
