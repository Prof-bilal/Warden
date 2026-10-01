// Package packs supplies reviewed candidate profiles without broad default grants.
package packs

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/warden-sandbox/warden/internal/policy"
	"gopkg.in/yaml.v3"
)

//go:embed catalog.json
var catalog []byte

type Pack struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Category string   `json:"category"`
	Upstream string   `json:"upstream"`
	Artifact string   `json:"artifact"`
	Version  string   `json:"version"`
	Profile  string   `json:"profile"`
	Status   string   `json:"status"`
	Reviewed string   `json:"reviewed"`
	PathMode string   `json:"pathMode"`
	Hosts    []string `json:"hosts"`
	Env      []string `json:"env"`
	Note     string   `json:"note"`
}

func List() []Pack {
	var result []Pack
	if err := json.Unmarshal(catalog, &result); err != nil {
		panic(err)
	}
	return result
}

func Find(id string) (Pack, error) {
	for _, p := range List() {
		if p.ID == id {
			return p, nil
		}
	}
	return Pack{}, fmt.Errorf("unknown pack %q; use warden packs list", id)
}

// Generate grants only user-selected data and prepared runtime directories.
// It never installs or launches an upstream, and never copies credential values.
func Generate(id, dataPath, runtimePath string) ([]byte, error) {
	pack, err := Find(id)
	if err != nil {
		return nil, err
	}
	runtimePath, err = narrowDirectory(runtimePath)
	if err != nil {
		return nil, fmt.Errorf("runtime: %w", err)
	}
	p := policy.Policy{Filesystem: policy.Filesystem{Read: []string{runtimePath}}, Network: policy.Network{Allow: pack.Hosts}, Env: policy.Env{Allow: pack.Env}, Limits: policy.Limits{MemoryMB: 512, TimeoutS: 3600}}
	if pack.PathMode != "none" {
		dataPath, err = narrowDirectory(dataPath)
		if err != nil {
			return nil, fmt.Errorf("data: %w", err)
		}
		if pack.PathMode == "write" {
			if policy.Within(dataPath, runtimePath) {
				return nil, fmt.Errorf("storage must be separate from the runtime directory")
			}
			rel, e := filepath.Rel(dataPath, runtimePath)
			if e == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return nil, fmt.Errorf("storage must not contain the runtime directory")
			}
			p.Filesystem.Write = []string{dataPath}
		} else {
			p.Filesystem.Read = append(p.Filesystem.Read, dataPath)
		}
	} else if dataPath != "" {
		return nil, fmt.Errorf("%s does not require a data directory", id)
	}
	p.Normalize()
	if err := p.Validate(); err != nil {
		return nil, err
	}
	b, err := yaml.Marshal(p)
	if err != nil {
		return nil, err
	}
	header := fmt.Sprintf("# Warden candidate pack: %s (%s@%s)\n# Reviewed: %s; upstream/runtime workflow not certified.\n# %s\n# Runtime must be prepared separately; no dependency download grants.\n", pack.ID, pack.Artifact, pack.Version, pack.Reviewed, pack.Note)
	return append([]byte(header), b...), nil
}

func narrowDirectory(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("an explicit existing directory is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("must be a directory")
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		if h, e := filepath.EvalSymlinks(home); e == nil {
			home = h
		}
	}
	if abs == home || filepath.Dir(abs) == abs {
		return "", fmt.Errorf("whole-home and filesystem-root grants are refused")
	}
	for _, broad := range []string{"/home", "/Users", "/usr", "/etc", "/var", "/opt"} {
		if abs == broad {
			return "", fmt.Errorf("select a narrower directory")
		}
	}
	return abs, nil
}

func Digest(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }

// PreparedCommand validates the local npm package manifest and selects its
// installed entry point. Dependency preparation writes a package-lock; no npx
// cache or online package resolution is needed for execution.
func PreparedCommand(id, runtimePath, dataPath string) ([]string, error) {
	pack, err := Find(id)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(pack.Artifact, "@") {
		return nil, nil
	}
	runtimePath, err = narrowDirectory(runtimePath)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(runtimePath, "node_modules", filepath.FromSlash(pack.Artifact))
	raw, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("prepare %s@%s in the runtime directory first", pack.Artifact, pack.Version)
	}
	var metadata struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if json.Unmarshal(raw, &metadata) != nil || metadata.Name != pack.Artifact || metadata.Version != pack.Version {
		return nil, fmt.Errorf("installed package identity/version differs from the reviewed profile")
	}
	entry := filepath.Join(root, "dist", "index.js")
	if _, err = os.Stat(entry); err != nil {
		return nil, fmt.Errorf("prepared entry point is missing")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		return nil, fmt.Errorf("Node.js is required")
	}
	node, err = filepath.Abs(node)
	if err != nil {
		return nil, err
	}
	argv := []string{node, entry}
	if id == "filesystem" {
		dataPath, err = narrowDirectory(dataPath)
		if err != nil {
			return nil, err
		}
		argv = append(argv, dataPath)
	}
	if id == "brave" {
		argv = append(argv, "--transport", "stdio")
	}
	return argv, nil
}
