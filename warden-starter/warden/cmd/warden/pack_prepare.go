package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/warden-sandbox/warden/internal/packs"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/privatefile"
	"github.com/warden-sandbox/warden/internal/sandbox"
)

// prepareNPMRuntime installs one pinned npm artifact (name@version) into a
// NEW directory dest inside the sandbox: registry-only egress, install
// scripts disabled, isolated HOME and npm configuration. It performs no
// post-install validation; callers verify the installed identity.
func prepareNPMRuntime(artifact, dest, backend string) error {
	return installNPMRuntime(artifact, dest, backend, nil, "", false)
}

func installNPMRuntime(spec, dest, backend string, archive []byte, scriptManifest string, offline bool) error {
	if _, e := sandbox.Resolve(backend); e != nil {
		return e
	}
	npm, e := exec.LookPath("npm")
	if e != nil {
		return fmt.Errorf("npm is required for isolated preparation")
	}
	npm, e = filepath.Abs(npm)
	if e != nil {
		return e
	}
	realNPM, e := filepath.EvalSymlinks(npm)
	if e != nil {
		return e
	}
	if ext := strings.ToLower(filepath.Ext(realNPM)); ext == ".cmd" || ext == ".ps1" || ext == ".bat" {
		realNPM = filepath.Join(filepath.Dir(npm), "node_modules", "npm", "bin", "npm-cli.js")
		if _, e = os.Stat(realNPM); e != nil {
			return fmt.Errorf("cannot locate npm-cli.js beside the Windows launcher")
		}
	}
	node, e := exec.LookPath("node")
	if e != nil {
		return fmt.Errorf("node.js is required")
	}
	node, e = filepath.Abs(node)
	if e != nil {
		return e
	}
	dest, e = filepath.Abs(dest)
	if e != nil {
		return e
	}
	// A new directory guarantees no user files or credential configuration can
	// be overwritten. Failures retain this directory for diagnosis, not reuse.
	if e = os.Mkdir(dest, 0700); e != nil {
		return fmt.Errorf("runtime output must be a new directory with an existing parent: %w", e)
	}
	if e = privatefile.Protect(dest, true); e != nil {
		return e
	}
	var archiveDir string
	artifact := spec
	if archive != nil || runtime.GOOS == "windows" {
		archiveDir, e = os.MkdirTemp(filepath.Dir(dest), ".warden-archive-")
		if e != nil {
			return e
		}
		defer os.RemoveAll(archiveDir)
		if e = privatefile.Protect(archiveDir, true); e != nil {
			return e
		}
	}
	if archive != nil {
		artifact = filepath.Join(archiveDir, "reviewed.tgz")
		if e = writeExclusive(artifact, archive); e != nil {
			return e
		}
	}
	configDir := dest
	if runtime.GOOS == "windows" {
		// Keep these private files outside dest so normalization retains their explicit read grants.
		configDir = archiveDir
	}
	config := filepath.Join(configDir, ".npmrc")
	if e = writeExclusive(config, []byte{}); e != nil {
		return e
	}
	globalConfig := filepath.Join(configDir, "global.npmrc")
	if e = writeExclusive(globalConfig, []byte{}); e != nil {
		return e
	}
	path := filepath.Dir(node) + string(os.PathListSeparator) + "/usr/bin"
	if runtime.GOOS == "windows" {
		// Node's explicit lifecycle runner needs Windows' command interpreter.
		path = filepath.Dir(node) + string(os.PathListSeparator) + filepath.Join(os.Getenv("SystemRoot"), "System32")
	}
	vars := map[string]string{"PATH": path, "NPM_CONFIG_CACHE": filepath.Join(dest, ".npm-cache"), "NPM_CONFIG_USERCONFIG": config, "NPM_CONFIG_GLOBALCONFIG": globalConfig}
	vars["HOME"] = dest // an isolated installer home, never the user's home
	if runtime.GOOS == "windows" {
		vars["USERPROFILE"] = dest
		vars["TEMP"] = dest
		vars["TMP"] = dest
	}
	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{filepath.Dir(node), filepath.Dir(filepath.Dir(realNPM))}, Write: []string{dest}},
		Network:    policy.Network{Allow: []string{"registry.npmjs.org"}},
		Limits:     policy.Limits{MemoryMB: 1024, TimeoutS: 600},
	}
	if runtime.GOOS == "windows" {
		// These protected file ACLs do not inherit grants on their parent.
		p.Filesystem.Read = append(p.Filesystem.Read, config, globalConfig)
		if archive != nil {
			p.Filesystem.Read = append(p.Filesystem.Read, artifact)
		}
	} else if archiveDir != "" {
		p.Filesystem.Read = append(p.Filesystem.Read, archiveDir)
	}
	// Preparation can precede a stdio MCP session, which reserves stdout.
	args := []string{node, realNPM, "install", "--silent", "--ignore-scripts", "--no-audit", "--no-fund", "--package-lock=true", "--prefix", dest}
	if offline {
		p.Network.Allow = nil
		args = append(args, "--offline")
	}
	args = append(args, "--", artifact)
	if err := runInstallSandboxed(args, p, backend, vars); err != nil {
		return err
	}
	if scriptManifest == "" {
		return nil
	}
	commands, headers, err := reviewedNPMLifecycle(spec, dest, node, realNPM, scriptManifest)
	if err != nil {
		return err
	}
	p.Network.Allow = nil
	if headers != "" {
		p.Filesystem.Read = append(p.Filesystem.Read, headers)
		if runtime.GOOS == "linux" {
			// Make launches its recipe shell by absolute path.
			p.Filesystem.Read = append(p.Filesystem.Read, "/bin/sh")
		}
	}
	for _, command := range commands {
		// A previous hook may have rewritten the package manifest.
		if _, err := readReviewedNPMManifest(spec, dest, scriptManifest); err != nil {
			return err
		}
		if err := runInstallSandboxed(command, p, backend, vars); err != nil {
			return fmt.Errorf("approved lifecycle step for %q: %w", spec, err)
		}
	}
	return nil
}

func reviewedNPMLifecycle(spec, dest, node, npm, expectedManifest string) ([][]string, string, error) {
	pkg, err := readReviewedNPMManifest(spec, dest, expectedManifest)
	if err != nil {
		return nil, "", err
	}
	var commands [][]string
	var headers string
	for _, hook := range []string{"preinstall", "install", "postinstall"} {
		if pkg.Scripts[hook] != "" {
			// An explicit run-script with --ignore-scripts runs the named hook only.
			commands = append(commands, []string{node, npm, "run-script", "--ignore-scripts", "--workspaces=false", "--global=false", "--prefix", pkg.Dir, "--", hook})
		} else if hook == "install" && pkg.Scripts["preinstall"] == "" && (pkg.GYPFile == nil || *pkg.GYPFile) {
			binding, err := os.Lstat(filepath.Join(pkg.Dir, "binding.gyp"))
			if err != nil && !os.IsNotExist(err) {
				return nil, "", err
			}
			if err == nil {
				if !binding.Mode().IsRegular() {
					return nil, "", fmt.Errorf("approved binding.gyp is not a regular file")
				}
				realNode, err := filepath.EvalSymlinks(node)
				if err != nil {
					return nil, "", err
				}
				nodeRoot := filepath.Dir(filepath.Dir(realNode))
				headers = filepath.Join(nodeRoot, "include", "node")
				if _, err := os.Stat(filepath.Join(headers, "node.h")); err != nil {
					return nil, "", fmt.Errorf("approved native build requires local headers for the selected Node.js installation")
				}
				gyp := filepath.Join(filepath.Dir(filepath.Dir(npm)), "node_modules", "node-gyp", "bin", "node-gyp.js")
				commands = append(commands, []string{node, gyp, "--directory", pkg.Dir, "--nodedir", nodeRoot, "rebuild"})
			}
		}
	}
	return commands, headers, nil
}

type npmLifecycleManifest struct {
	Name, Version string
	Scripts       map[string]string
	GYPFile       *bool  `json:"gypfile"`
	Dir           string `json:"-"`
}

func readReviewedNPMManifest(spec, dest, expectedManifest string) (npmLifecycleManifest, error) {
	var pkg npmLifecycleManifest
	name, version, ok := splitNPMSpec(spec)
	if !ok || version == "" {
		return pkg, fmt.Errorf("lifecycle approval requires an exact package version")
	}
	runtimeDir, err := filepath.EvalSymlinks(dest)
	if err != nil {
		return pkg, err
	}
	root := filepath.Join(runtimeDir, "node_modules", filepath.FromSlash(name))
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || !policy.Within(resolved, runtimeDir) {
		return pkg, fmt.Errorf("approved package is outside its prepared runtime")
	}
	manifestPath := filepath.Join(root, "package.json")
	info, err := os.Lstat(manifestPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return pkg, fmt.Errorf("approved package manifest is not a bounded regular file")
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != expectedManifest {
		return pkg, fmt.Errorf("installed manifest differs from the reviewed archive")
	}
	if err := json.Unmarshal(raw, &pkg); err != nil || pkg.Name != name || pkg.Version != version {
		return pkg, fmt.Errorf("installed package does not match the approved name and version")
	}
	pkg.Dir = root
	return pkg, nil
}

func runInstallSandboxed(args []string, p policy.Policy, backend string, vars map[string]string) error {
	code, e := runSandboxWithEnv(args, p, backend, vars)
	if e != nil || code != 0 {
		return fmt.Errorf("isolated preparation failed (exit %d); incomplete output retained; no direct retry: %v", code, e)
	}
	return nil
}

func cmdPackPrepare(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("pack ID required")
	}
	pack, e := packs.Find(args[0])
	if e != nil {
		return e
	}
	if !strings.HasPrefix(pack.Artifact, "@") {
		return fmt.Errorf("isolated prepare currently covers npm profiles; prepare Git/Python or the GitHub native binary separately")
	}
	fs := flag.NewFlagSet("packs prepare", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	output := fs.String("output", "", "new dedicated runtime directory")
	backend := fs.String("backend", "auto", "installation sandbox")
	if e = fs.Parse(args[1:]); e != nil || fs.NArg() != 0 || *output == "" {
		return fmt.Errorf("usage: warden packs prepare <npm-pack> --output <new-directory> [--backend auto]")
	}
	if !validSetupBackend(*backend) {
		return fmt.Errorf("invalid backend")
	}
	if e = prepareNPMRuntime(pack.Artifact+"@"+pack.Version, *output, *backend); e != nil {
		return e
	}
	dest, e := filepath.Abs(*output)
	if e != nil {
		return e
	}
	if _, e = packs.PreparedCommand(pack.ID, dest, func() string {
		if pack.ID == "filesystem" {
			return dest
		}
		return ""
	}()); e != nil {
		return e
	}
	fmt.Fprintln(os.Stderr, "Prepared pinned npm package inside the sandbox with install scripts disabled. Review the generated package-lock before runtime use.")
	return nil
}
