// Package gateway integrates Warden with MCP gateway / client configs.
//
// It parses the two config families used across the MCP ecosystem:
//
//  1. Claude-style JSON: {"mcpServers": {"name": {"command","args","env","cwd","type","url"}}}
//     (also accepts the VS Code alias "servers"). This is what Claude Desktop,
//     Claude Code, Cursor, Windsurf, Docker MCP Gateway clients, and the
//     lasso-security mcp-gateway ("servers" nested under its own entry) use.
//  2. Gateway YAML registries:
//     a. jonfairbanks/mcp-gateway style: {upstreams: [{id, transport, command,
//     args, env, cwd, endpoint}]}
//     b. MikkoParkkola/mcp-gateway style: {backends: {name: {command, args,
//     env, cwd, http_url/url}}}
//
// Normalization: every entry becomes a ServerEntry. Only stdio entries (a
// local command to spawn) can be sandboxed. Remote entries (type/url,
// http_url, endpoint, non-stdio transport) are marked Remote and are skipped
// by init/run with a clear reason — Warden sandboxes local processes, it does
// not proxy remote URLs.
//
// Security invariants (see AGENTS.md):
//   - Deny by default: generated starter policies grant no filesystem paths
//     and no network hosts. The operator must widen them explicitly (e.g. via
//     `warden trace` + `warden init`) after review.
//   - Policy values are never synthesized from gateway secrets: only env var
//     NAMES become env.allow; values always come from the parent environment
//     at runtime.
//   - Fail closed: missing files, unknown entries, unresolvable commands, or
//     a missing per-server policy is an error, never an unsandboxed run.
package gateway

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/warden-sandbox/warden/internal/policy"
	"gopkg.in/yaml.v3"
)

// ServerEntry is one MCP server normalized from any supported config format.
type ServerEntry struct {
	// Name is the server key (mcpServers key, upstream id, backend name).
	Name string
	// Command is the full argv (command + args) for stdio servers.
	Command []string
	// Env holds literal KEY=VALUE pairs from the gateway config. Values are
	// kept in memory only for `wrap` output; generated policies store NAMES
	// only (never values).
	Env map[string]string
	// Cwd is the working directory hint, if the config sets one.
	Cwd string
	// Remote is true for SSE/HTTP servers (url/endpoint/http_url or an
	// explicit non-stdio type). Remote servers have no local process to
	// sandbox and are skipped with Reason set.
	Remote bool
	// URL is the remote endpoint for Remote entries (informational).
	URL string
	// Reason explains why a Remote entry cannot be sandboxed.
	Reason string
	// Source records which config family the entry came from
	// ("mcpServers", "upstreams", "backends").
	Source string
}

// Config is a parsed gateway file: the detected format plus all entries.
type Config struct {
	// Path is the source file (for error messages).
	Path string
	// Format is "json" or "yaml".
	Format string
	// Servers are sorted by name for deterministic output.
	Servers []ServerEntry
}

// Stdio returns only the locally-spawned (sandboxable) entries.
func (c *Config) Stdio() []ServerEntry {
	var out []ServerEntry
	for _, s := range c.Servers {
		if !s.Remote {
			out = append(out, s)
		}
	}
	return out
}

// Find returns the entry with the given name or an error listing known names.
func (c *Config) Find(name string) (ServerEntry, error) {
	for _, s := range c.Servers {
		if s.Name == name {
			return s, nil
		}
	}
	return ServerEntry{}, fmt.Errorf("unknown server %q (known: %s)", name, c.Names(", "))
}

// Names returns all server names sorted.
func (c *Config) Names(sep string) string {
	names := make([]string, len(c.Servers))
	for i, s := range c.Servers {
		names[i] = s.Name
	}
	return strings.Join(names, sep)
}

// Load reads path and parses it as JSON (Claude-style) or YAML (gateway
// registry) based on extension (.json → JSON, anything else → YAML with a
// JSON fallback, since JSON is valid YAML but our YAML schema differs).
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read gateway config %q: %w", path, err)
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".json" || ext == ".jsonc" {
		return parseJSON(path, data)
	}
	// YAML first; if the document has an mcpServers/servers mapping, treat it
	// as JSON-in-YAML (some gateways ship .yaml with the Claude shape).
	cfg, err := parseYAML(path, data)
	if err != nil {
		return nil, err
	}
	if len(cfg.Servers) == 0 {
		if jc, jerr := parseJSON(path, data); jerr == nil && len(jc.Servers) > 0 {
			return jc, nil
		}
	}
	return cfg, nil
}

// --- JSON (Claude-style) ---

type jsonServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
	Cwd     string            `json:"cwd"`
	Type    string            `json:"type"`
	URL     string            `json:"url"`
}

type jsonDoc struct {
	MCPServers map[string]jsonServer `json:"mcpServers"`
	Servers    map[string]jsonServer `json:"servers"` // VS Code alias
}

func parseJSON(path string, data []byte) (*Config, error) {
	var doc jsonDoc
	dec := json.NewDecoder(strings.NewReader(string(data)))
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse gateway config %q: invalid JSON: %w", path, err)
	}
	// Merge both keys; mcpServers wins on collision.
	merged := make(map[string]jsonServer, len(doc.MCPServers)+len(doc.Servers))
	for k, v := range doc.Servers {
		merged[k] = v
	}
	for k, v := range doc.MCPServers {
		merged[k] = v
	}
	if len(merged) == 0 {
		return nil, fmt.Errorf("parse gateway config %q: no \"mcpServers\" (or \"servers\") entries found", path)
	}
	cfg := &Config{Path: path, Format: "json"}
	for name, js := range merged {
		e, err := jsonEntry(name, js)
		if err != nil {
			return nil, fmt.Errorf("gateway config %q server %q: %w", path, name, err)
		}
		cfg.Servers = append(cfg.Servers, e)
	}
	sort.Slice(cfg.Servers, func(i, j int) bool { return cfg.Servers[i].Name < cfg.Servers[j].Name })
	return cfg, nil
}

func jsonEntry(name string, js jsonServer) (ServerEntry, error) {
	if name == "" {
		return ServerEntry{}, fmt.Errorf("server name must not be empty")
	}
	typ := strings.ToLower(strings.TrimSpace(js.Type))
	url := strings.TrimSpace(js.URL)
	if typ == "http" || typ == "sse" || typ == "streamable-http" || typ == "streamable_http" || url != "" {
		if url == "" {
			url = "(no url given)"
		}
		return ServerEntry{
			Name: name, Remote: true, URL: url, Source: "mcpServers",
			Reason: fmt.Sprintf("server %q is remote (type/url %q); no local process to sandbox", name, url),
		}, nil
	}
	if strings.TrimSpace(js.Command) == "" {
		return ServerEntry{}, fmt.Errorf("stdio server %q has no command", name)
	}
	cmd := append([]string{js.Command}, js.Args...)
	env := make(map[string]string, len(js.Env))
	for k, v := range js.Env {
		if strings.TrimSpace(k) == "" || strings.ContainsAny(k, " \t\n=") {
			return ServerEntry{}, fmt.Errorf("invalid env var name %q", k)
		}
		env[k] = v
	}
	return ServerEntry{Name: name, Command: cmd, Env: env, Cwd: js.Cwd, Source: "mcpServers"}, nil
}

// --- YAML (gateway registries) ---

type yamlDoc struct {
	Upstreams []yamlUpstream      `yaml:"upstreams"`
	Backends  map[string]yamlBack `yaml:"backends"`
	// Tolerate Claude-shape-in-YAML without failing: handled via fallback.
	MCPServers map[string]jsonServer `yaml:"mcpServers"`
	ServersMap map[string]jsonServer `yaml:"servers"`
}

type yamlUpstream struct {
	ID        string            `yaml:"id"`
	Name      string            `yaml:"name"`
	Transport string            `yaml:"transport"`
	Command   yamlCommand       `yaml:"command"`
	Args      []string          `yaml:"args"`
	Env       map[string]string `yaml:"env"`
	Cwd       string            `yaml:"cwd"`
	Endpoint  string            `yaml:"endpoint"`
	URL       string            `yaml:"url"`
}

type yamlBack struct {
	Command     string            `yaml:"command"`
	Args        []string          `yaml:"args"`
	Env         map[string]string `yaml:"env"`
	Cwd         string            `yaml:"cwd"`
	HTTPURL     string            `yaml:"http_url"`
	URL         string            `yaml:"url"`
	Description string            `yaml:"description"`
}

// yamlCommand accepts `command: "npx ..."` (shell-split on spaces — only for
// simple cases; prefer a list) or `command: ["npx", "-y", ...]`.
type yamlCommand []string

func (c *yamlCommand) UnmarshalYAML(value *yaml.Node) error {
	var single string
	if err := value.Decode(&single); err == nil {
		*c = strings.Fields(single)
		return nil
	}
	var list []string
	if err := value.Decode(&list); err != nil {
		return fmt.Errorf("command must be a string or list of strings")
	}
	*c = list
	return nil
}

func parseYAML(path string, data []byte) (*Config, error) {
	var doc yamlDoc
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(false) // gateway files carry many unrelated keys; ignore them
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse gateway config %q: invalid YAML: %w", path, err)
	}
	cfg := &Config{Path: path, Format: "yaml"}
	for _, u := range doc.Upstreams {
		e, err := upstreamEntry(u)
		if err != nil {
			return nil, fmt.Errorf("gateway config %q upstream %q: %w", path, u.ID, err)
		}
		cfg.Servers = append(cfg.Servers, e)
	}
	for name, b := range doc.Backends {
		e, err := backendEntry(name, b)
		if err != nil {
			return nil, fmt.Errorf("gateway config %q backend %q: %w", path, name, err)
		}
		cfg.Servers = append(cfg.Servers, e)
	}
	// Seen map guards against a file mixing registry + Claude shapes.
	seen := make(map[string]bool, len(cfg.Servers))
	for _, s := range cfg.Servers {
		seen[s.Name] = true
	}
	for name, js := range doc.MCPServers {
		if seen[name] {
			continue
		}
		e, err := jsonEntry(name, js)
		if err != nil {
			return nil, fmt.Errorf("gateway config %q server %q: %w", path, name, err)
		}
		cfg.Servers = append(cfg.Servers, e)
	}
	if len(cfg.Servers) == 0 {
		return nil, fmt.Errorf("parse gateway config %q: no upstreams, backends, or mcpServers entries found", path)
	}
	sort.Slice(cfg.Servers, func(i, j int) bool { return cfg.Servers[i].Name < cfg.Servers[j].Name })
	return cfg, nil
}

func upstreamEntry(u yamlUpstream) (ServerEntry, error) {
	name := u.ID
	if name == "" {
		name = u.Name
	}
	if name == "" {
		return ServerEntry{}, fmt.Errorf("upstream entry has neither id nor name")
	}
	t := strings.ToLower(strings.TrimSpace(u.Transport))
	if t == "" {
		t = "stdio"
	}
	url := u.Endpoint
	if url == "" {
		url = u.URL
	}
	if t != "stdio" || url != "" {
		if url == "" {
			url = "(transport " + t + ")"
		}
		return ServerEntry{
			Name: name, Remote: true, URL: url, Source: "upstreams",
			Reason: fmt.Sprintf("server %q is remote (transport %q); no local process to sandbox", name, t),
		}, nil
	}
	if len(u.Command) == 0 {
		return ServerEntry{}, fmt.Errorf("stdio upstream %q has no command", name)
	}
	cmd := append(append([]string{}, u.Command...), u.Args...)
	if err := checkEnv(u.Env); err != nil {
		return ServerEntry{}, err
	}
	return ServerEntry{Name: name, Command: cmd, Env: u.Env, Cwd: u.Cwd, Source: "upstreams"}, nil
}

func backendEntry(name string, b yamlBack) (ServerEntry, error) {
	if name == "" {
		return ServerEntry{}, fmt.Errorf("backend name must not be empty")
	}
	url := b.HTTPURL
	if url == "" {
		url = b.URL
	}
	if url != "" {
		return ServerEntry{
			Name: name, Remote: true, URL: url, Source: "backends",
			Reason: fmt.Sprintf("server %q is remote (http_url %q); no local process to sandbox", name, url),
		}, nil
	}
	var cmd []string
	if strings.TrimSpace(b.Command) != "" {
		// Mikko-style backends use a single shell string ("npx -y ...").
		cmd = strings.Fields(b.Command)
	}
	cmd = append(cmd, b.Args...)
	if len(cmd) == 0 {
		return ServerEntry{}, fmt.Errorf("stdio backend %q has no command", name)
	}
	if err := checkEnv(b.Env); err != nil {
		return ServerEntry{}, err
	}
	return ServerEntry{Name: name, Command: cmd, Env: b.Env, Cwd: b.Cwd, Source: "backends"}, nil
}

func checkEnv(env map[string]string) error {
	for k := range env {
		if strings.TrimSpace(k) == "" || strings.ContainsAny(k, " \t\n=") {
			return fmt.Errorf("invalid env var name %q", k)
		}
	}
	return nil
}

// StarterPolicy returns a deny-by-default policy for a stdio entry.
//
// The command's executable is resolved to an absolute path via exec.LookPath
// when possible (the sandbox requires absolute paths). Filesystem and network
// grants are empty — the operator widens them after `warden trace`. Env
// allowlists the NAMES from the gateway entry; values are never copied.
func StarterPolicy(e ServerEntry) (policy.Policy, error) {
	if e.Remote {
		return policy.Policy{}, fmt.Errorf("cannot sandbox remote server %q: %s", e.Name, e.Reason)
	}
	if len(e.Command) == 0 {
		return policy.Policy{}, fmt.Errorf("server %q has no command", e.Name)
	}
	cmd := append([]string{}, e.Command...)
	if !filepath.IsAbs(cmd[0]) {
		if abs, err := exec.LookPath(cmd[0]); err == nil {
			cmd[0] = abs
		}
	}
	var envNames []string
	for k := range e.Env {
		envNames = append(envNames, k)
	}
	sort.Strings(envNames)
	p := policy.Policy{
		Command:    cmd,
		Filesystem: policy.Filesystem{},
		Network:    policy.Network{},
		Env:        policy.Env{Allow: envNames},
	}
	if err := p.Validate(); err != nil {
		return policy.Policy{}, fmt.Errorf("generated policy for %q is invalid: %w", e.Name, err)
	}
	return p, nil
}

// PolicyPath returns the per-server policy file for name inside dir.
func PolicyPath(dir, name string) string {
	return filepath.Join(dir, name+".yaml")
}

// WritePolicies generates one deny-by-default policy per stdio server into
// dir (created if missing). Existing files are never overwritten. Remote
// servers are skipped. Returns the written and skipped server names.
func WritePolicies(cfg *Config, dir string) (written, skipped []string, err error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create policies dir %q: %w", dir, err)
	}
	for _, e := range cfg.Servers {
		if e.Remote {
			skipped = append(skipped, e.Name+" (remote: not sandboxable)")
			continue
		}
		p, err := StarterPolicy(e)
		if err != nil {
			return written, skipped, err
		}
		data, err := p.Marshal()
		if err != nil {
			return written, skipped, fmt.Errorf("encode policy for %q: %w", e.Name, err)
		}
		out := PolicyPath(dir, e.Name)
		f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			if os.IsExist(err) {
				skipped = append(skipped, e.Name+" (policy already exists)")
				continue
			}
			return written, skipped, fmt.Errorf("create %q: %w", out, err)
		}
		header := fmt.Sprintf("# Warden gateway starter policy for server %q\n# Source: %s (%s)\n# Deny-by-default: widen filesystem/network after `warden trace`. Review env.allow below.\n", e.Name, cfg.Path, e.Source)
		if _, err := f.Write(append([]byte(header), data...)); err != nil {
			f.Close()
			return written, skipped, fmt.Errorf("write %q: %w", out, err)
		}
		if err := f.Close(); err != nil {
			return written, skipped, fmt.Errorf("close %q: %w", out, err)
		}
		written = append(written, e.Name)
	}
	return written, skipped, nil
}

// WrappedCommand returns the argv that runs a stdio server through Warden:
// [wardenBin run --policy <policies/name.yaml> [--backend B] -- <cmd...>].
func WrappedCommand(wardenBin, policyPath, backend string, e ServerEntry) []string {
	out := []string{wardenBin, "run", "--policy", policyPath}
	if backend != "" && backend != "auto" {
		out = append(out, "--backend", backend)
	}
	out = append(out, "--")
	return append(out, e.Command...)
}
