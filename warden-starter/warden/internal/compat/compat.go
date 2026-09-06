// Package compat is the M8 compatibility-matrix regression harness.
//
// testdata/compat/matrix.yaml is the single source of truth: one entry per
// real-world MCP server with its verdict, failure class, and grant probes.
// The tests in compat_test.go enforce that every entry has a matching
// fixture policy and that every fixture policy grants exactly what the
// manifest claims, using only the policy engine (no bwrap, no network) so
// they run deterministically in CI. End-to-end backend runs remain manual;
// see testdata/compat/README.md.
package compat

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Entry mirrors one verdicts[] item in matrix.yaml.
type Entry struct {
	Name           string   `yaml:"name"`
	Upstream       string   `yaml:"upstream"`
	Verdict        string   `yaml:"verdict"`
	FailureClass   string   `yaml:"failure_class"`
	FixNote        string   `yaml:"fix_note"`
	Policy         string   `yaml:"policy"`
	Notes          string   `yaml:"notes"`
	ProbeAllowFile []string `yaml:"probe_allow_files"`
	ProbeDenyFile  []string `yaml:"probe_deny_files"`
	ProbeAllowHost []string `yaml:"probe_allow_hosts"`
	ProbeAllowEnv  []string `yaml:"probe_allow_env"`
}

// Manifest is the parsed matrix.yaml.
type Manifest struct {
	Verdicts []Entry `yaml:"verdicts"`
}

// Dir returns the absolute path to testdata/compat.
func Dir() (string, error) {
	dir := filepath.Join("..", "..", "testdata", "compat")
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve compat dir: %w", err)
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("compat dir %s: %w", abs, err)
	}
	return abs, nil
}

// LoadManifest reads and parses matrix.yaml from dir.
func LoadManifest(dir string) (Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "matrix.yaml"))
	if err != nil {
		return Manifest{}, fmt.Errorf("read matrix.yaml: %w", err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse matrix.yaml: %w", err)
	}
	return m, nil
}
