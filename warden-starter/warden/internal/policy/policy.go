// Package policy parses, validates, and normalizes the Warden policy
// schema. A Policy is the single source of truth for what a sandboxed
// process may access; the sandbox backend translates it into
// backend-specific primitives.
//
// Deny by default: any path, host, or environment variable that is not
// explicitly granted here is blocked by the sandbox.
package policy

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Policy is the parsed, validated form of a policy YAML file.
//
// Fields correspond one-to-one with the schema documented in
// ARCHITECTURE.md. All filesystem paths are absolute after ResolvePaths.
type Policy struct {
	// Command is how to start the sandboxed server. Optional in the file
	// when the CLI supplies it via `warden run --policy <file> -- <cmd...>`.
	Command []string `yaml:"command,omitempty"`

	// Filesystem grants read-only and read-write paths inside the sandbox.
	Filesystem Filesystem `yaml:"filesystem"`

	// Network grants hostnames the sandbox may connect to.
	Network Network `yaml:"network"`

	// Env lists environment variable names passed through from the parent.
	// Values are never stored in the policy file.
	Env Env `yaml:"env"`

	// Limits constrains address-space use and wall-clock execution time.
	Limits Limits `yaml:"limits"`
}

// Filesystem is the filesystem section of a policy.
type Filesystem struct {
	// Read lists paths mounted read-only inside the sandbox.
	Read []string `yaml:"read,omitempty"`
	// Write lists paths mounted read-write inside the sandbox.
	Write []string `yaml:"write,omitempty"`
}

// Network is the network section of a policy.
type Network struct {
	// Allow lists hostnames the sandboxed process may reach.
	Allow []string `yaml:"allow,omitempty"`
}

// Env is the environment section of a policy.
type Env struct {
	// Allow lists environment variable names to pass through.
	Allow []string `yaml:"allow,omitempty"`
}

// Limits is the resource-limits section of a policy. Parsed for schema
// compatibility; enforcement is M3.
type Limits struct {
	MemoryMB int `yaml:"memory_mb,omitempty"`
	TimeoutS int `yaml:"timeout_s,omitempty"`
}

// Load reads and parses a policy file from path, resolves relative paths
// against the policy's directory, and validates the result.
func Load(path string) (Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("read policy %q: %w", path, err)
	}

	var p Policy
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // fail closed on unknown/typo'd keys
	if err := dec.Decode(&p); err != nil {
		return Policy{}, fmt.Errorf("parse policy %q: %w", path, err)
	}

	dir := filepath.Dir(path)
	if err := p.ResolvePaths(dir); err != nil {
		return Policy{}, fmt.Errorf("policy %q: %w", path, err)
	}
	p.Normalize()
	if err := p.Validate(); err != nil {
		return Policy{}, fmt.Errorf("policy %q: %w", path, err)
	}
	return p, nil
}

// ValidateRunnable checks the constraints that only apply once a command
// source is known. Load intentionally does not call this: the CLI may
// supply the command after `--`, so a policy file without `command:` is
// valid on its own.
func (p *Policy) ValidateRunnable(cmd []string) error {
	if len(cmd) == 0 {
		return fmt.Errorf("no command configured: run `warden run --policy <file> -- <command...>` or set `command` in the policy")
	}
	return nil
}

// ResolvePaths resolves all relative filesystem grants against dir,
// producing absolute paths inside the sandbox. A grant's visible path in
// the sandbox is the same absolute path as on the host, so a relative grant
// "./data" in /home/user/proj/app becomes /home/user/proj/app/data both on
// the host and inside the sandbox.
//
// Because the sandbox is deny-by-default, the resolved path is what the
// sandboxed process sees — it does not leak the path's host prefix beyond
// the granted paths themselves.
func (p *Policy) ResolvePaths(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve policy directory: %w", err)
	}
	for i, path := range p.Filesystem.Read {
		r, err := resolve(absDir, path)
		if err != nil {
			return fmt.Errorf("filesystem.read[%d]: %w", i, err)
		}
		p.Filesystem.Read[i] = r
	}
	for i, path := range p.Filesystem.Write {
		r, err := resolve(absDir, path)
		if err != nil {
			return fmt.Errorf("filesystem.write[%d]: %w", i, err)
		}
		p.Filesystem.Write[i] = r
	}
	return nil
}

func resolve(dir, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path must not be empty")
	}
	cleaned := filepath.Clean(path)
	abs := cleaned
	if !filepath.IsAbs(cleaned) {
		abs = filepath.Join(dir, cleaned)
	}
	// A path that resolves outside the policy directory is an error: the
	// policy author intends a sibling or parent path, which the sandbox
	// should not silently grant.
	if abs == "." || abs == string(filepath.Separator) {
		return "", fmt.Errorf("path %q resolves to filesystem root; refusing to grant it", path)
	}
	return abs, nil
}

// Validate checks semantic constraints that parsing cannot. It must be
// called after ResolvePaths (so grants are absolute) and before the policy
// is handed to a sandbox backend.
func (p *Policy) Validate() error {
	for _, w := range p.Filesystem.Write {
		if err := checkGrantPath(w, "write"); err != nil {
			return err
		}
	}
	for _, r := range p.Filesystem.Read {
		if err := checkGrantPath(r, "read"); err != nil {
			return err
		}
	}
	for i, host := range p.Network.Allow {
		if err := validateHost(host); err != nil {
			return fmt.Errorf("network.allow[%d]: %w", i, err)
		}
	}
	for _, name := range p.Env.Allow {
		if name == "" || strings.ContainsAny(name, " \t\n=") {
			return fmt.Errorf("env.allow: %q is not a valid environment variable name", name)
		}
	}
	if p.Limits.MemoryMB < 0 {
		return fmt.Errorf("limits.memory_mb must not be negative")
	}
	if uint64(p.Limits.MemoryMB) > ^uint64(0)/(1024*1024) {
		return fmt.Errorf("limits.memory_mb is too large")
	}
	if p.Limits.TimeoutS < 0 {
		return fmt.Errorf("limits.timeout_s must not be negative")
	}
	if uint64(p.Limits.TimeoutS) > uint64((time.Duration(1<<63-1))/time.Second) {
		return fmt.Errorf("limits.timeout_s is too large")
	}
	return nil
}

func checkGrantPath(path, kind string) error {
	if path == "" {
		return fmt.Errorf("filesystem.%s: path must not be empty", kind)
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("filesystem.%s: %q must be absolute", kind, path)
	}
	return nil
}

// validateHost accepts a bare hostname or an IP literal. Ports and URLs are
// not accepted: the policy grants hosts, and the egress layer handles ports.
func validateHost(host string) error {
	if host == "" {
		return fmt.Errorf("host must not be empty")
	}
	// IP literal (IPv4 or IPv6) — allowed as-is.
	if ip := net.ParseIP(host); ip != nil {
		return nil
	}
	if strings.ContainsAny(host, "/?#") {
		return fmt.Errorf("%q is not a bare hostname or IP", host)
	}
	// A colon now means either a port or a malformed v6-ish literal; both
	// are rejected (IPv6 literals were accepted by net.ParseIP above).
	if strings.Contains(host, ":") {
		return fmt.Errorf("%q is not a bare hostname or IP", host)
	}
	name := host
	if strings.HasSuffix(name, ".") || len(name) > 253 {
		return fmt.Errorf("%q is not a valid hostname", host)
	}
	for _, label := range strings.Split(name, ".") {
		if err := validateLabel(label); err != nil {
			return fmt.Errorf("%q: %w", host, err)
		}
	}
	return nil
}

func validateLabel(label string) error {
	if len(label) == 0 || len(label) > 63 {
		return fmt.Errorf("label %q is empty or too long", label)
	}
	if label[0] == '-' || label[len(label)-1] == '-' {
		return fmt.Errorf("label %q must not start or end with a hyphen", label)
	}
	for _, c := range label {
		if !(c == '-' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return fmt.Errorf("label %q has invalid character %q", label, c)
		}
	}
	return nil
}

// ResolveCommand returns the command to run: the CLI-specified command
// wins; otherwise the policy's command is used. Returns an error if neither
// is set. cli is nil or empty when no command was given on the CLI.
func (p *Policy) ResolveCommand(cli []string) ([]string, error) {
	if len(cli) > 0 {
		return cli, nil
	}
	if len(p.Command) > 0 {
		return p.Command, nil
	}
	return nil, fmt.Errorf("no command: pass it after `--` or set `command:` in the policy")
}

// ResolveExecutable resolves cmd[0] to an absolute path when it is a bare
// name (e.g. "npx", "uvx", "node") via exec.LookPath, mirroring what
// gateway.StarterPolicy already does for gateway-registered servers. Most
// public MCP servers are installed and launched this way, so compat testing
// (M8) showed the bare "must be an absolute path" failure is the single
// most common first-run friction.
//
// Resolution is fail-closed: an unresolvable name returns an error instead
// of a guess, and an already-absolute path is returned untouched (existence
// is checked by the CLI/backend, not here). Callers must still pass the
// resolved command to the sandbox backend, which requires absolute paths so
// it can bind-mount the executable's parent directory.
func ResolveExecutable(cmd []string) ([]string, error) {
	if len(cmd) == 0 {
		return nil, fmt.Errorf("no command configured: run `warden run --policy <file> -- <command...>` or set `command` in the policy")
	}
	if filepath.IsAbs(cmd[0]) {
		return cmd, nil
	}
	abs, err := exec.LookPath(cmd[0])
	if err != nil {
		return nil, fmt.Errorf("command %q is not an absolute path and was not found on PATH (run `command -v %s` to locate it, then use the full path)", cmd[0], cmd[0])
	}
	out := append([]string{abs}, cmd[1:]...)
	return out, nil
}

// EnvAllowlist returns a copy of the names to pass through, with duplicates
// removed and in stable order.
func (p *Policy) EnvAllowlist() []string {
	seen := make(map[string]bool)
	var out []string
	for _, name := range p.Env.Allow {
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

// UnimplementedAdvisories returns human-readable warnings for policy
// sections that are parsed but not yet enforced by this build.
func (p *Policy) UnimplementedAdvisories() []string {
	var out []string
	return out
}
