package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Matrix is the declarative case list under testdata/egress/matrix.yaml.
type Matrix struct {
	Version int    `yaml:"version"`
	Cases   []Case `yaml:"cases"`
}

// Case is one row of the network-path matrix. Most cases are single probe
// invocations whose outcome is judged by the generic evaluator; a small set
// of Kind values switch to runner-side orchestration.
type Case struct {
	ID       string   `yaml:"id"`
	Group    string   `yaml:"group"`
	Title    string   `yaml:"title"`
	Kind     string   `yaml:"kind,omitempty"`
	Backends []string `yaml:"backends,omitempty"`
	Policy   string   `yaml:"policy,omitempty"`
	Probe    []string `yaml:"probe,omitempty"`
	Steps    []Step   `yaml:"steps,omitempty"`
	Expect   string   `yaml:"expect,omitempty"`
	Checks   []string `yaml:"checks,omitempty"`
	Requires []string `yaml:"requires,omitempty"`
	Timeout  string   `yaml:"timeout,omitempty"`
	Reason   string   `yaml:"reason,omitempty"`
	Notes    string   `yaml:"notes,omitempty"`

	// AllowCollector must receive a token hit for allow cases.
	AllowCollector string `yaml:"allow_collector,omitempty"`
	// Forbidden collectors must have zero token hits. When empty, the
	// runner's default forbidden set is used.
	Forbidden []string `yaml:"forbidden,omitempty"`
	// Accept extends the acceptable error classes for deny cases.
	Accept []string `yaml:"accept,omitempty"`
	// AcceptStatus extends the acceptable HTTP status codes for deny cases.
	AcceptStatus []int `yaml:"accept_status,omitempty"`
	// Extra carries string parameters used by named checks.
	Extra map[string]string `yaml:"extra,omitempty"`
}

// Step is one probe invocation for multi-step cases.
type Step struct {
	Name     string   `yaml:"name,omitempty"`
	Probe    []string `yaml:"probe"`
	Expect   string   `yaml:"expect,omitempty"`
	Checks   []string `yaml:"checks,omitempty"`
	Required bool     `yaml:"required,omitempty"`
}

// Name returns the step's stable evidence name.
func (s Step) Named(index int) string {
	if s.Name != "" {
		return s.Name
	}
	return fmt.Sprintf("step-%d", index+1)
}

func loadMatrix(path string) (*Matrix, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Matrix
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if m.Version != 1 {
		return nil, fmt.Errorf("%s: unsupported matrix version %d", path, m.Version)
	}
	seen := map[string]bool{}
	for i := range m.Cases {
		c := &m.Cases[i]
		if c.ID == "" {
			return nil, fmt.Errorf("%s: case %d has no id", path, i)
		}
		if seen[c.ID] {
			return nil, fmt.Errorf("%s: duplicate case id %q", path, c.ID)
		}
		seen[c.ID] = true
		if c.Expect == "" && c.Kind == "" {
			return nil, fmt.Errorf("%s: case %s has neither expect nor kind", path, c.ID)
		}
	}
	return &m, nil
}

// substitute replaces {{name}} placeholders from vars. Unknown placeholders
// are left intact so a typo shows up in evidence instead of silently
// becoming an empty string.
func substitute(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}

func substituteAll(list []string, vars map[string]string) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = substitute(s, vars)
	}
	return out
}
