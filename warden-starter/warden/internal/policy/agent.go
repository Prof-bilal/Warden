package policy

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// Agent enables an explicit sanitized workspace session on a native host
// backend (Linux bubblewrap, macOS Seatbelt, or Windows AppContainer/WFP).
type Agent struct {
	Workspace  string     `yaml:"workspace"`
	Redaction  Redaction  `yaml:"redaction"`
	SecretExec SecretExec `yaml:"secret_exec,omitempty"`
}
type Redaction struct {
	Env   []string       `yaml:"env,omitempty"`
	Files []RedactedFile `yaml:"files,omitempty"`
	Apply string         `yaml:"apply,omitempty"`
}
type RedactedFile struct {
	Path   string `yaml:"path"`
	Format string `yaml:"format"`
	// Fields are exact, slash-separated paths: /credentials/api_key.
	// Escape ~ as ~0 and / as ~1. Dotenv fields are variable names.
	Fields []string `yaml:"fields"`
}
type SecretExec struct {
	Profiles map[string]SecretProfile `yaml:"profiles,omitempty"`
}

// SecretProfile is a fixed HTTPS GET operation performed by the supervisor.
// Arbitrary agent-written commands never receive the credential.
type SecretProfile struct {
	URL string `yaml:"url"`
	Env string `yaml:"env"`
}

func (a *Agent) resolve(dir string) error {
	var err error
	a.Workspace, err = resolve(dir, a.Workspace)
	return err
}
func (a Agent) validate(p Policy) error {
	if p.MCP != nil {
		return fmt.Errorf("agent workspace policies cannot be used as MCP proxy policies")
	}
	if !filepath.IsAbs(a.Workspace) {
		return fmt.Errorf("agent.workspace must be absolute after resolution")
	}
	for _, base := range []string{"/usr", "/lib", "/lib64", "/proc", "/dev"} {
		if a.Workspace == base || Within(a.Workspace, base) {
			return fmt.Errorf("agent workspace cannot overlap sandbox runtime paths")
		}
	}
	if a.Redaction.Apply != "" && a.Redaction.Apply != "review" {
		return fmt.Errorf("agent.redaction.apply must be review")
	}
	grants := append(append([]string{}, p.Filesystem.Read...), p.Filesystem.Write...)
	if len(grants) != 1 || grants[0] != a.Workspace {
		return fmt.Errorf("agent mode requires exactly one filesystem grant, equal to agent.workspace")
	}
	seen := map[string]bool{}
	for _, f := range a.Redaction.Files {
		if f.Path == "" || filepath.IsAbs(f.Path) || filepath.Clean(f.Path) != f.Path || f.Path == "." || strings.HasPrefix(f.Path, ".."+string(filepath.Separator)) || f.Path == ".." || strings.Split(f.Path, string(filepath.Separator))[0] == ".warden" || strings.Split(f.Path, string(filepath.Separator))[0] == ".git" {
			return fmt.Errorf("agent redaction path must be a clean workspace-relative file")
		}
		if seen[f.Path] {
			return fmt.Errorf("duplicate agent redaction file")
		}
		seen[f.Path] = true
		switch f.Format {
		case "dotenv", "json", "yaml", "toml":
		default:
			return fmt.Errorf("unsupported agent redaction format")
		}
		if len(f.Fields) == 0 {
			return fmt.Errorf("agent redaction fields cannot be empty")
		}
		fields := map[string]bool{}
		for _, field := range f.Fields {
			if field == "" || fields[field] {
				return fmt.Errorf("empty or duplicate redaction field")
			}
			fields[field] = true
			if f.Format != "dotenv" && !strings.HasPrefix(field, "/") {
				return fmt.Errorf("structured fields use JSON Pointer paths, e.g. /credentials/api_key")
			}
		}
	}
	for _, name := range a.Redaction.Env {
		if name == "" || strings.ContainsAny(name, "=\x00\r\n \t") {
			return fmt.Errorf("invalid agent redaction environment name")
		}
	}
	for name, profile := range a.SecretExec.Profiles {
		for _, ch := range name {
			if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_') {
				return fmt.Errorf("secret profile names use letters, numbers, - or _")
			}
		}
		u, err := url.Parse(profile.URL)
		if name == "" || err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || (u.Port() != "" && u.Port() != "443") {
			return fmt.Errorf("secret profile requires a fixed HTTPS URL without userinfo, query, fragment, or custom port")
		}
		if profile.Env == "" || strings.ContainsAny(profile.Env, "=\x00\r\n \t") {
			return fmt.Errorf("invalid secret profile environment name")
		}
		allowed := false
		for _, host := range p.Network.Allow {
			if host == u.Hostname() {
				allowed = true
			}
		}
		if !allowed {
			return fmt.Errorf("secret profile host must be explicitly listed in network.allow")
		}
	}
	return nil
}
