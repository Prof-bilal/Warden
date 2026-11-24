// Package autopilot compiles a reviewed project enrollment into a native
// agent policy. Repository metadata and advisor output cannot add grants.
package autopilot

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/warden-sandbox/warden/internal/policy"
)

const (
	ReadOnly      = "read-only"
	ReviewedEdits = "reviewed-edits"
	revision      = 1
)

type Hint struct {
	Marker  string `json:"marker"`
	Project string `json:"project"`
	Build   string `json:"build_hint,omitempty"`
	Test    string `json:"test_hint,omitempty"`
}

var markers = []Hint{
	{"go.mod", "Go", "go build ./...", "go test ./..."},
	{"Cargo.toml", "Rust / Cargo", "cargo build", "cargo test"},
	{"pyproject.toml", "Python", "", ""},
	{"uv.lock", "Python / uv", "", "uv run pytest"},
	{"requirements.txt", "Python / pip", "", ""},
	{"package.json", "JavaScript / TypeScript", "", ""},
	{"package-lock.json", "npm", "npm run build", "npm test"},
	{"pnpm-lock.yaml", "pnpm", "pnpm build", "pnpm test"},
	{"yarn.lock", "Yarn", "yarn build", "yarn test"},
	{"bun.lock", "Bun", "bun run build", "bun test"},
	{"Makefile", "Make", "", ""},
	{".mcp.json", "MCP configuration", "", ""},
	{"opencode.json", "OpenCode configuration", "", ""},
	{"opencode.jsonc", "OpenCode configuration", "", ""},
	{"AGENTS.md", "Agent instructions", "", ""},
	{"CLAUDE.md", "Claude instructions", "", ""},
}

type Project struct {
	Root  string   `json:"root"`
	OS    string   `json:"os"`
	Arch  string   `json:"arch"`
	Hints []Hint   `json:"hints"`
	Tools []string `json:"tools_on_path"`
}

// Inspect checks fixed names in the project root without opening their contents.
func Inspect(root string) (Project, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return Project{}, err
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return Project{}, err
	}
	dir, err := os.OpenRoot(canonical)
	if err != nil {
		return Project{}, err
	}
	defer dir.Close()
	p := Project{Root: canonical, OS: runtime.GOOS, Arch: runtime.GOARCH, Hints: []Hint{}, Tools: []string{}}
	for _, hint := range markers {
		info, err := dir.Lstat(hint.Marker)
		switch {
		case os.IsNotExist(err):
		case err != nil:
			return Project{}, err
		case info.Mode().IsRegular():
			p.Hints = append(p.Hints, hint)
		}
	}
	for _, name := range []string{"bwrap", "claude", "codex", "opencode", "git", "go", "cargo", "python3", "uv", "node", "npm", "pnpm", "yarn", "bun", "make"} {
		if _, err := exec.LookPath(name); err == nil {
			p.Tools = append(p.Tools, name)
		}
	}
	return p, nil
}

type Enrollment struct {
	Revision int    `json:"revision"`
	Project  string `json:"project"`
	Program  string `json:"program"`
	Mode     string `json:"mode"`
}

func Enroll(project, command, mode string) (Enrollment, error) {
	program, err := resolveProgram(command)
	if err != nil {
		return Enrollment{}, err
	}
	e := Enrollment{Revision: revision, Project: project, Program: program, Mode: mode}
	_, err = e.Policy(project)
	return e, err
}

func (e Enrollment) Policy(project string) (policy.Policy, error) {
	if e.Revision != revision || !filepath.IsAbs(project) || filepath.Clean(project) != project || filepath.Dir(project) == project || e.Project != project {
		return policy.Policy{}, fmt.Errorf("enrollment does not match this project and policy revision")
	}
	if !filepath.IsAbs(e.Program) || filepath.Clean(e.Program) != e.Program || overlap(e.Program, project) || policy.IsProtectedSecretPath(e.Program) {
		return policy.Policy{}, fmt.Errorf("agent executable must be an installed program outside the project and protected paths")
	}
	p := policy.Policy{
		Command:    []string{e.Program},
		Agent:      &policy.Agent{Workspace: project, Redaction: policy.Redaction{Apply: "review"}},
		Filesystem: policy.Filesystem{ProtectSecrets: true},
		Limits:     policy.Limits{MemoryMB: 2048, TimeoutS: 3600},
	}
	for _, name := range []string{".mcp.json", ".claude", ".codex", ".opencode", "opencode.json", "opencode.jsonc"} {
		p.Filesystem.Deny = append(p.Filesystem.Deny, filepath.Join(project, name))
	}
	switch e.Mode {
	case ReadOnly:
		p.Filesystem.Read = []string{project}
	case ReviewedEdits:
		p.Filesystem.Write = []string{project}
	default:
		return policy.Policy{}, fmt.Errorf("unknown workspace access mode")
	}
	return p, p.Validate()
}

// VerifyProgram refuses a replaced symlink instead of resolving a new target
// under an existing approval. Normal in-place upgrades at the same path remain
// the responsibility of the trusted host installer.
func (e Enrollment) VerifyProgram() error {
	path, err := resolveProgram(e.Program)
	if err != nil {
		return err
	}
	if path != e.Program {
		return fmt.Errorf("installed agent path changed; revoke and reapprove")
	}
	return nil
}

func resolveProgram(command string) (string, error) {
	path, err := exec.LookPath(command)
	if err != nil {
		return "", fmt.Errorf("locate installed agent: %w", err)
	}
	return filepath.EvalSymlinks(path)
}

func overlap(a, b string) bool {
	return a == b || policy.Within(a, b) || policy.Within(b, a)
}
