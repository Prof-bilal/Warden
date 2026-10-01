package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/warden-sandbox/warden/internal/packs"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/privatefile"
	"github.com/warden-sandbox/warden/internal/sandbox"
)

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
	if _, e = sandbox.Resolve(*backend); e != nil {
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
		return fmt.Errorf("Node.js is required")
	}
	node, e = filepath.Abs(node)
	if e != nil {
		return e
	}
	dest, e := filepath.Abs(*output)
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
	config := filepath.Join(dest, ".npmrc")
	if e = writeExclusive(config, []byte{}); e != nil {
		return e
	}
	globalConfig := filepath.Join(dest, "global.npmrc")
	if e = writeExclusive(globalConfig, []byte{}); e != nil {
		return e
	}
	vars := map[string]string{"PATH": filepath.Dir(node) + string(os.PathListSeparator) + "/usr/bin", "NPM_CONFIG_CACHE": filepath.Join(dest, ".npm-cache"), "NPM_CONFIG_USERCONFIG": config, "NPM_CONFIG_GLOBALCONFIG": globalConfig}
	vars["HOME"] = dest // an isolated installer home, never the user's home
	// sandbox.Run uses the parent env; only these explicitly configured names
	// are allowed through, with no HOME or user credential configuration.
	restore := map[string]*string{}
	for key, value := range vars {
		if v, ok := os.LookupEnv(key); ok {
			saved := v
			restore[key] = &saved
		} else {
			restore[key] = nil
		}
		if e = os.Setenv(key, value); e != nil {
			return e
		}
	}
	restoreEnvironment := func() {
		for key, value := range restore {
			if value == nil {
				_ = os.Unsetenv(key)
			} else {
				_ = os.Setenv(key, *value)
			}
		}
	}
	defer restoreEnvironment()
	p := policy.Policy{Filesystem: policy.Filesystem{Read: []string{filepath.Dir(node), filepath.Dir(filepath.Dir(realNPM))}, Write: []string{dest}}, Network: policy.Network{Allow: []string{"registry.npmjs.org"}}, Env: policy.Env{Allow: []string{"PATH", "HOME", "NPM_CONFIG_CACHE", "NPM_CONFIG_USERCONFIG", "NPM_CONFIG_GLOBALCONFIG"}}, Limits: policy.Limits{MemoryMB: 1024, TimeoutS: 600}}
	p.Normalize()
	if e = p.Validate(); e != nil {
		return e
	}
	// Invoke the trusted npm JS entry point through the selected Node runtime;
	// no shebang/PATH-dependent resolution of the installer is necessary.
	code, e := sandbox.Run([]string{node, realNPM, "install", "--ignore-scripts", "--no-audit", "--no-fund", "--package-lock=true", "--prefix", dest, pack.Artifact + "@" + pack.Version}, p, *backend)
	if e != nil || code != 0 {
		return fmt.Errorf("isolated preparation failed (exit %d); incomplete output retained; no direct retry: %v", code, e)
	}
	restoreEnvironment()
	if _, e = packs.PreparedCommand(pack.ID, dest, func() string {
		if pack.ID == "filesystem" {
			return dest
		}
		return ""
	}()); e != nil {
		return e
	}
	fmt.Fprintln(os.Stderr, "Prepared pinned npm package inside the sandbox; install scripts were disabled. Review the generated package-lock before runtime use. Profile remains candidate.")
	return nil
}
