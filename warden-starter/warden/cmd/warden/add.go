package main

// warden add connects one MCP server to one client in a single command:
//
//	warden add slack to claude-desktop
//	warden add @scope/server@1.2.3 to claude-code --allow-host api.example.com
//
// It prepares the runtime inside the sandbox, generates a deny-by-default
// policy, registers a wrapped launcher in the client configuration, and
// reports required credentials by name (never values). The whole flow is
// previewed and confirmed before anything is written; --yes skips the
// prompt and --dry-run never writes. Undo: warden unwrap --client <id>
// --server <entry>.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/warden-sandbox/warden/internal/clientconfig"
	"github.com/warden-sandbox/warden/internal/packs"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/sandbox"
	"github.com/warden-sandbox/warden/internal/ui"
	"gopkg.in/yaml.v3"
)

type addOptions struct {
	server, client, name, scope, config, backend, runtime, data, policyPath string
	allowHosts, allowEnv, allowRead, allowWrite                             []string
	yes, dryRun                                                             bool
}

func parseAddArgs(args []string) (addOptions, error) {
	var o addOptions
	o.backend = "auto"
	o.scope = "project"
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return o, fmt.Errorf("usage: warden add <server> to <client> [options]")
	}
	o.server = args[0]
	rest := args[1:]
	if len(rest) > 0 && rest[0] == "to" {
		if len(rest) < 2 {
			return o, fmt.Errorf("'to' must be followed by a client id; run warden clients")
		}
		o.client = rest[1]
		rest = rest[2:]
	}
	for i := 0; i < len(rest); i++ {
		arg := rest[i]
		value := func(flag string) (string, error) {
			if i+1 >= len(rest) {
				return "", fmt.Errorf("flag %s requires a value", flag)
			}
			i++
			return rest[i], nil
		}
		switch {
		case arg == "--client":
			v, e := value(arg)
			if e != nil {
				return o, e
			}
			if o.client != "" {
				return o, fmt.Errorf("--client specified twice")
			}
			o.client = v
		case arg == "--name":
			v, e := value(arg)
			if e != nil {
				return o, e
			}
			o.name = v
		case arg == "--scope":
			v, e := value(arg)
			if e != nil {
				return o, e
			}
			o.scope = v
		case arg == "--config":
			v, e := value(arg)
			if e != nil {
				return o, e
			}
			o.config = v
		case arg == "--backend":
			v, e := value(arg)
			if e != nil {
				return o, e
			}
			o.backend = v
		case arg == "--runtime":
			v, e := value(arg)
			if e != nil {
				return o, e
			}
			o.runtime = v
		case arg == "--data":
			v, e := value(arg)
			if e != nil {
				return o, e
			}
			o.data = v
		case arg == "--policy":
			v, e := value(arg)
			if e != nil {
				return o, e
			}
			o.policyPath = v
		case arg == "--allow-host":
			v, e := value(arg)
			if e != nil {
				return o, e
			}
			o.allowHosts = append(o.allowHosts, v)
		case arg == "--allow-env":
			v, e := value(arg)
			if e != nil {
				return o, e
			}
			o.allowEnv = append(o.allowEnv, v)
		case arg == "--allow-read":
			v, e := value(arg)
			if e != nil {
				return o, e
			}
			o.allowRead = append(o.allowRead, v)
		case arg == "--allow-write":
			v, e := value(arg)
			if e != nil {
				return o, e
			}
			o.allowWrite = append(o.allowWrite, v)
		case arg == "--yes" || arg == "-y":
			o.yes = true
		case arg == "--dry-run":
			o.dryRun = true
		default:
			return o, fmt.Errorf("unknown flag %q; use warden help add", arg)
		}
	}
	if o.client == "" {
		return o, fmt.Errorf("missing client: use 'warden add <server> to <client>' or --client")
	}
	if o.yes && o.dryRun {
		return o, fmt.Errorf("--yes and --dry-run cannot be combined")
	}
	return o, nil
}

// npmNameRe matches npm package names (optionally @scope/name); the version
// suffix is split off before this check.
var npmNameRe = regexp.MustCompile(`^(@[a-z0-9-._~]+/)?[a-z0-9-._~]+$`)

// npmVersionRe accepts common semver-ish and range specs (1.2.3, ^2.0, next).
var npmVersionRe = regexp.MustCompile(`^[0-9A-Za-z._+~^*-]+$`)

// splitNPMSpec splits an npm spec into package name and optional version.
// It reports ok=false for strings that are clearly not npm package names
// (pack IDs, file paths, URLs).
func splitNPMSpec(s string) (name, version string, ok bool) {
	if s == "" {
		return "", "", false
	}
	rest := s
	scoped := strings.HasPrefix(rest, "@")
	if scoped {
		slash := strings.Index(rest, "/")
		if slash < 0 {
			return "", "", false
		}
		name = rest[:slash+1] // keep the slash so "@scope" + "server" rejoins
		rest = rest[slash+1:]
	}
	if at := strings.LastIndex(rest, "@"); at > 0 {
		version = rest[at+1:]
		rest = rest[:at]
	}
	if rest == "" || !scoped && strings.ContainsAny(rest, "/\\ \t:@") {
		return "", "", false
	}
	name += rest
	if !npmNameRe.MatchString(name) {
		return "", "", false
	}
	if version != "" && !npmVersionRe.MatchString(version) {
		return "", "", false
	}
	return name, version, true
}

// sanitizeEntryName derives a client config entry name from a package name.
func sanitizeEntryName(s string) string {
	base := s
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	var b strings.Builder
	for _, r := range strings.ToLower(base) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "mcp-server"
	}
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}

// wardenDataHome mirrors XDG_DATA_HOME semantics for runtime storage.
func wardenDataHome() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" && filepath.IsAbs(v) {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".local", "share")
}

// defaultRuntimeDir is where `add` keeps prepared runtimes and policies.
func defaultRuntimeDir(kind, id string) string {
	return filepath.Join(wardenDataHome(), "warden", "runtimes", kind, id)
}

// preparedEntry reports whether the runtime directory already holds a usable
// installation, and the entry point when it does.
func preparedEntry(runtimeDir, pkgRoot string) (string, bool) {
	info, err := os.Lstat(runtimeDir)
	if err != nil || !info.IsDir() {
		return "", false
	}
	entry, err := packs.ResolveEntry(pkgRoot)
	if err != nil {
		return "", false
	}
	return entry, true
}

// addSource carries what resolveAddSource produced for a catalog pack or
// generic npm package.
type addSource struct {
	policyBody []byte
	upstream   []string
	envNames   []string
	label      string
}

// resolveAddSource turns the requested server into a policy body, upstream
// command, credential names, and a display description. Catalog packs get
// their reviewed grants; other npm packages get a deny-by-default starter
// policy extended by the --allow-* flags.
func resolveAddSource(o addOptions) (src addSource, runtimeDir string, err error) {
	if pack, err := packs.Find(o.server); err == nil {
		if !strings.HasPrefix(pack.Artifact, "@") {
			return src, "", fmt.Errorf("pack %q needs a manually prepared runtime; see warden packs prepare", pack.ID)
		}
		runtimeDir := o.runtime
		if runtimeDir == "" {
			runtimeDir = defaultRuntimeDir(pack.ID, "default")
		}
		if _, ok := preparedEntry(runtimeDir, filepath.Join(runtimeDir, "node_modules", filepath.FromSlash(pack.Artifact))); !ok {
			if _, err := os.Lstat(runtimeDir); err == nil {
				return src, "", fmt.Errorf("runtime directory %s exists but is not a prepared %s@%s install; remove it or pass --runtime", runtimeDir, pack.Artifact, pack.Version)
			}
			fmt.Fprintf(os.Stderr, "Preparing %s@%s inside the sandbox (install scripts disabled)...\n", pack.Artifact, pack.Version)
			if err := prepareNPMRuntime(pack.Artifact+"@"+pack.Version, runtimeDir, o.backend); err != nil {
				return src, "", err
			}
		}
		dataPath := o.data
		if pack.PathMode != "none" && dataPath == "" {
			dataPath = filepath.Join(wardenDataHome(), "warden", "data", pack.ID)
			if err := os.MkdirAll(dataPath, 0700); err != nil {
				return src, "", err
			}
		}
		argv, err := packs.PreparedCommand(pack.ID, runtimeDir, dataPath)
		if err != nil {
			return src, "", err
		}
		body, err := packs.Generate(pack.ID, dataPath, runtimeDir)
		if err != nil {
			return src, "", err
		}
		encoded, _ := json.Marshal(argv)
		src.policyBody = append([]byte("command: "+string(encoded)+"\n"), body...)
		src.upstream = argv
		src.envNames = pack.Env
		src.label = fmt.Sprintf("%s (%s@%s)", pack.Name, pack.Artifact, pack.Version)
		return src, runtimeDir, nil
	}
	name, version, ok := splitNPMSpec(o.server)
	if !ok {
		return src, "", fmt.Errorf("unknown server %q; use a pack id from warden packs list, or an npm package like @scope/mcp-server@1.2.3", o.server)
	}
	spec := name
	if version != "" {
		spec += "@" + version
	} else {
		fmt.Fprintln(os.Stderr, "note: no version pinned; installing the latest release. Prefer name@version for reproducible grants.")
	}
	runtimeDir = o.runtime
	if runtimeDir == "" {
		runtimeDir = defaultRuntimeDir("npm", sanitizeEntryName(name))
	}
	node, err := exec.LookPath("node")
	if err != nil {
		return src, "", fmt.Errorf("Node.js is required")
	}
	if node, err = filepath.Abs(node); err != nil {
		return src, "", err
	}
	entry, ok := preparedEntry(runtimeDir, filepath.Join(runtimeDir, "node_modules", filepath.FromSlash(name)))
	if !ok {
		if _, err := os.Lstat(runtimeDir); err == nil {
			return src, "", fmt.Errorf("runtime directory %s exists but holds no usable install; remove it or pass --runtime", runtimeDir)
		}
		fmt.Fprintf(os.Stderr, "Preparing %s inside the sandbox (install scripts disabled)...\n", spec)
		if err := prepareNPMRuntime(spec, runtimeDir, o.backend); err != nil {
			return src, "", err
		}
		if entry, err = packs.ResolveEntry(filepath.Join(runtimeDir, "node_modules", filepath.FromSlash(name))); err != nil {
			return src, "", err
		}
	}
	// Deny-by-default starter policy: the runtime directory is readable so
	// the entry point can load; everything else comes from --allow-* flags.
	runtimeAbs, err := filepath.Abs(runtimeDir)
	if err != nil {
		return src, "", err
	}
	pol := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{runtimeAbs}},
		Network:    policy.Network{Allow: o.allowHosts},
		Env:        policy.Env{Allow: o.allowEnv},
		Limits:     policy.Limits{MemoryMB: 512, TimeoutS: 300},
	}
	for _, p := range o.allowRead {
		abs, err := filepath.Abs(p)
		if err != nil {
			return src, "", err
		}
		pol.Filesystem.Read = append(pol.Filesystem.Read, abs)
	}
	for _, w := range o.allowWrite {
		abs, err := filepath.Abs(w)
		if err != nil {
			return src, "", err
		}
		pol.Filesystem.Write = append(pol.Filesystem.Write, abs)
	}
	pol.Normalize()
	if err := pol.Validate(); err != nil {
		return src, "", err
	}
	body, err := yaml.Marshal(pol)
	if err != nil {
		return src, "", err
	}
	argv := []string{node, entry}
	encoded, _ := json.Marshal(argv)
	src.policyBody = append([]byte("command: "+string(encoded)+"\n"), body...)
	src.upstream = argv
	src.envNames = o.allowEnv
	src.label = spec + " (deny-by-default starter grants)"
	return src, runtimeDir, nil
}

// resolveBYOPolicy handles --policy: the caller's reviewed policy replaces
// the generated one, and its command becomes the upstream.
func resolveBYOPolicy(o addOptions) (string, []string, error) {
	p, err := policy.Load(o.policyPath)
	if err != nil {
		return "", nil, err
	}
	if len(p.Command) == 0 {
		return "", nil, fmt.Errorf("--policy must include a command; it becomes the sandboxed launcher")
	}
	return o.policyPath, p.Command, nil
}

// writeAddPolicy persists the generated policy, refusing to silently replace
// an existing one with different grants.
func writeAddPolicy(path string, body []byte) error {
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) != string(body) {
			return fmt.Errorf("policy %s already exists with different grants; remove it or pass --policy to use your own", path)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return writeExclusive(path, body)
}

// reportCredentials prints credential status by name only, never values.
func reportCredentials(envNames []string) {
	var missing, present []string
	for _, name := range envNames {
		if name == "" || name == "HOME" {
			continue
		}
		if os.Getenv(name) == "" {
			missing = append(missing, name)
		} else {
			present = append(present, name)
		}
	}
	for _, name := range present {
		fmt.Fprintf(os.Stderr, "  %s %s is set\n", ui.Green(ui.CheckMark()), name)
	}
	for _, name := range missing {
		fmt.Fprintf(os.Stderr, "  %s export %s=...   (required; set it before starting the client)\n", ui.Dim(ui.CrossMark()), name)
	}
}

func cmdAdd(args []string) error {
	for _, a := range args {
		if a == "--list-clients" {
			if len(args) != 1 {
				return fmt.Errorf("--list-clients cannot be combined with other arguments")
			}
			return listClientConfigs(os.Stdout)
		}
	}
	o, err := parseAddArgs(args)
	if err != nil {
		return err
	}
	if !validSetupBackend(o.backend) {
		return fmt.Errorf("invalid backend")
	}
	c, err := clientconfig.Get(o.client)
	if err != nil {
		return err
	}
	if o.client == "claude-desktop" && o.scope == "project" {
		o.scope = "user"
	}
	configPath, err := clientconfig.ConfigPath(o.client, o.scope, o.config)
	if err != nil {
		return err
	}

	var policyPath string
	var upstream []string
	var label string
	var envNames []string
	if o.policyPath != "" {
		policyPath, upstream, err = resolveBYOPolicy(o)
		if err != nil {
			return err
		}
		label = o.server + " (own policy)"
	} else {
		var src addSource
		var runtimeDir string
		src, runtimeDir, err = resolveAddSource(o)
		if err != nil {
			return err
		}
		policyPath = filepath.Join(runtimeDir, "policy.yaml")
		if err := writeAddPolicy(policyPath, src.policyBody); err != nil {
			return err
		}
		upstream = src.upstream
		label = src.label
		envNames = src.envNames
	}

	entryName := o.name
	if entryName == "" {
		if pack, err := packs.Find(o.server); err == nil {
			entryName = pack.ID
		} else if name, _, ok := splitNPMSpec(o.server); ok {
			entryName = sanitizeEntryName(name)
		} else {
			entryName = sanitizeEntryName(o.server)
		}
	}
	if err := clientconfig.ValidEntryName(entryName); err != nil {
		return fmt.Errorf("--name: %v", err)
	}

	policyBytes, err := os.ReadFile(policyPath)
	if err != nil {
		return fmt.Errorf("policy failed validation: %v", err)
	}
	p, err := policy.LoadVerified(policyPath, clientconfig.Hash(policyBytes))
	if err != nil {
		return fmt.Errorf("policy failed validation: %v", err)
	}

	wardenBin, err := os.Executable()
	if err != nil {
		return err
	}
	if !filepath.IsAbs(wardenBin) {
		if wardenBin, err = filepath.Abs(wardenBin); err != nil {
			return err
		}
	}
	plan, err := clientconfig.PrepareCreate(configPath, c.Format, entryName, policyPath, wardenBin, o.backend, upstream)
	if err != nil {
		return err
	}

	// Preview: grants and paths only, never command arguments or secrets.
	state := "existing"
	if plan.ConfigMissing {
		state = "will be created"
	}
	fmt.Fprintf(os.Stderr, "\nAdd %s\n", label)
	fmt.Fprintf(os.Stderr, "  Client:  %s\n", c.Name)
	fmt.Fprintf(os.Stderr, "  Config:  %s (%s)\n", plan.Config, state)
	fmt.Fprintf(os.Stderr, "  Entry:   %s\n", entryName)
	fmt.Fprintf(os.Stderr, "  Policy:  %s (sha256 %s)\n", plan.Policy, shortHash(plan.PolicyHash))
	fmt.Fprintf(os.Stderr, "  Sandbox: warden run --policy ... --backend %s -- <upstream>\n", o.backend)
	grants, _ := json.Marshal(struct {
		Read     []string `json:"read"`
		Write    []string `json:"write"`
		Hosts    []string `json:"hosts"`
		EnvNames []string `json:"env_names"`
	}{p.Filesystem.Read, p.Filesystem.Write, p.Network.Allow, p.Env.Allow})
	fmt.Fprintf(os.Stderr, "  Grants:  %s\n", grants)
	if len(p.Network.Allow) == 0 {
		fmt.Fprintf(os.Stderr, "\n%s No network hosts are granted. Most servers need at least one:\n  --allow-host api.example.com (or refine later with warden trace + warden init)\n", ui.Dim("note:"))
	}
	if o.policyPath == "" {
		fmt.Fprintln(os.Stderr, "\nCredentials (names only; values never touch the config):")
		reportCredentials(envNames)
	}
	if o.dryRun {
		fmt.Fprintln(os.Stderr, "\nDry run: no files written, no processes started.")
		return nil
	}
	if !o.yes {
		if !ui.IsTerminalWriter(os.Stdin) || ui.IsCI() {
			return fmt.Errorf("refusing to apply without --yes in non-interactive mode")
		}
		fmt.Fprintf(os.Stderr, "\nApply this configuration? [y/N] ")
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		answer := strings.ToLower(strings.TrimSpace(line))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(os.Stderr, "Aborted; nothing was written.")
			return nil
		}
	}
	// Exercise the real sandbox with Warden's own inert probe before writing.
	probePolicy := policy.Policy{Limits: policy.Limits{TimeoutS: 10, MemoryMB: 128}}
	code, err := sandbox.Run([]string{wardenBin, "__setup-probe"}, probePolicy, o.backend)
	if err != nil || code != 0 {
		return fmt.Errorf("sandbox readiness probe failed; config remains unchanged: %v", err)
	}
	if _, err = clientconfig.Apply(plan); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "\n%s Added %s to %s. Restart the client, then verify an allowed task and a blocked task.\n", ui.Green(ui.CheckMark()), entryName, c.Name)
	fmt.Fprintf(os.Stderr, "Undo anytime: warden unwrap --client %s --server %s\n", o.client, entryName)
	return nil
}

// listClientConfigs prints every supported client adapter with its default
// configuration path and the current state of its server entries. Read-only:
// no server is started and no file is modified. Scopes without a default
// path ("explicit") need --config and are skipped.
func listClientConfigs(w io.Writer) error {
	home, _ := os.UserHomeDir()
	fmt.Fprintln(w, "Detected client configurations (read-only):")
	for _, c := range clientconfig.Clients {
		for _, scope := range c.Scopes {
			if scope == "explicit" {
				continue
			}
			path, err := clientconfig.ConfigPath(c.ID, scope, "")
			if err != nil {
				fmt.Fprintf(w, "  %-16s %-8s no default path  (%v)\n", c.ID, scope, err)
				continue
			}
			display := path
			if home != "" && strings.HasPrefix(path, home+string(filepath.Separator)) {
				display = "~" + path[len(home):]
			}
			if _, err := os.Lstat(path); err != nil {
				fmt.Fprintf(w, "  %-16s %-8s not configured  %s\n", c.ID, scope, display)
				continue
			}
			entries, err := clientconfig.Inventory(path, c.Format)
			if err != nil {
				// A config without an MCP servers section is normal (e.g.
				// ~/.claude.json holds other client state), not corruption.
				state := "unreadable"
				if strings.Contains(err.Error(), "mcp") {
					state = "no MCP servers section"
				}
				fmt.Fprintf(w, "  %-16s %-8s %-22s %s (%v)\n", c.ID, scope, state, display, err)
				continue
			}
			if len(entries) == 0 {
				fmt.Fprintf(w, "  %-16s %-8s empty          %s\n", c.ID, scope, display)
				continue
			}
			fmt.Fprintf(w, "  %-16s %-8s %s\n", c.ID, scope, display)
			for _, e := range entries {
				fmt.Fprintf(w, "    - %-24s %s\n", e.Name, e.State)
			}
		}
	}
	return nil
}

func shortHash(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}
