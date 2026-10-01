package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/warden-sandbox/warden/internal/clientconfig"
	"github.com/warden-sandbox/warden/internal/packs"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/privatefile"
	"github.com/warden-sandbox/warden/internal/sandbox"
)

func parsePolicyDigest(args []string) (string, []string, error) {
	var digest string
	out := []string{}
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			out = append(out, args[i:]...)
			break
		}
		value := ""
		if args[i] == "--policy-sha256" {
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--policy-sha256 requires a digest")
			}
			i++
			value = args[i]
		} else if strings.HasPrefix(args[i], "--policy-sha256=") {
			value = strings.TrimPrefix(args[i], "--policy-sha256=")
		} else {
			out = append(out, args[i])
			continue
		}
		raw, e := hex.DecodeString(value)
		if e != nil || len(raw) != 32 || digest != "" {
			return "", nil, fmt.Errorf("policy digest must be one SHA-256 value")
		}
		digest = strings.ToLower(value)
	}
	return digest, out, nil
}

func printEcosystemHelp(cmd string) {
	fmt.Fprintf(os.Stderr, "WARDEN %s\n\n", strings.ToUpper(cmd))
	switch cmd {
	case "packs":
		fmt.Fprintln(os.Stderr, `Usage:
  warden packs list [--json]
  warden packs show <pack>
  warden packs prepare <npm-pack> --output <new-runtime-directory> [--backend auto]
  warden packs generate <pack> --runtime <prepared-directory> [--path <data-directory>] --output <policy.yaml>

Profiles are candidates. prepare installs pinned npm releases inside the sandbox,
with install scripts disabled and registry-only egress. generate starts no servers.
The command writes a policy plus a metadata sidecar with its SHA-256 hash.`)
	case "clients":
		fmt.Fprintln(os.Stderr, "Usage: warden clients [--json]\nAdapters: claude-desktop, claude-code, cursor, codex, vscode, gemini, cline, cascade, generic")
	case "wrap":
		fmt.Fprintln(os.Stderr, `Usage:
  warden wrap --client <id> --server <existing-name> --policy <file> [--scope project|user] [--config <file>] [--backend auto] [--warden-bin <absolute-path>] [--use-policy-command] [--dry-run] [--yes] [--output <export-file>]

Default: redacted preview only. --yes applies after a sandbox readiness probe.
--dry-run performs no writes and starts no process. --output exports a private
config without changing the source. Apply creates an owner-only backup and
undo record. Remote entries are refused. No direct fallback is launched.
Restart your client and test an allowed task and a denied task afterward.
--server is a configured name, not a shell command or package name.`)
		fmt.Fprintln(os.Stderr, "--use-policy-command explicitly replaces upstream argv with the reviewed policy command. Undo keeps the original launcher.")
	case "unwrap":
		fmt.Fprintln(os.Stderr, "Usage: warden unwrap --client <id> --server <name> [--scope project|user] [--config <file>] [--dry-run] [--yes]\nDefault: preview only. Undo preserves unrelated edits and refuses launcher conflicts.")
	case "inventory":
		fmt.Fprintln(os.Stderr, "Usage: warden inventory --client <id> [--scope project|user] [--config <file>]\nRead-only stdio inventory. No server is started; wrapping is not workflow verification.")
	}
}

func cmdEcosystem(cmd string, args []string) error {
	switch cmd {
	case "packs":
		return cmdPacks(args)
	case "clients":
		if len(args) == 1 && args[0] == "--json" {
			b, _ := json.MarshalIndent(clientconfig.Clients, "", "  ")
			fmt.Println(string(b))
			return nil
		}
		if len(args) != 0 {
			return fmt.Errorf("usage: warden clients [--json]")
		}
		for _, c := range clientconfig.Clients {
			fmt.Printf("%-18s %-8s %s\n", c.ID, c.Format, strings.Join(c.Scopes, ", "))
		}
		return nil
	default:
		return cmdClientConfig(cmd, args)
	}
}

func cmdPacks(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: warden packs list|show|generate")
	}
	switch args[0] {
	case "prepare":
		return cmdPackPrepare(args[1:])
	case "list":
		if len(args) == 2 && args[1] == "--json" {
			b, _ := json.MarshalIndent(packs.List(), "", "  ")
			fmt.Println(string(b))
			return nil
		}
		if len(args) != 1 {
			return fmt.Errorf("usage: warden packs list [--json]")
		}
		for _, p := range packs.List() {
			fmt.Printf("%-12s %-12s %s\n", p.ID, p.Status, p.Profile)
		}
		return nil
	case "show":
		if len(args) != 2 {
			return fmt.Errorf("usage: warden packs show <pack>")
		}
		p, e := packs.Find(args[1])
		if e != nil {
			return e
		}
		b, _ := json.MarshalIndent(p, "", "  ")
		fmt.Println(string(b))
		return nil
	case "generate":
		if len(args) < 2 {
			return fmt.Errorf("pack ID is required")
		}
		fs := flag.NewFlagSet("packs generate", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		runtimeDir := fs.String("runtime", "", "prepared runtime directory")
		path := fs.String("path", "", "data directory")
		output := fs.String("output", "", "policy destination")
		if e := fs.Parse(args[2:]); e != nil {
			return fmt.Errorf("invalid pack options; use --help")
		}
		if fs.NArg() != 0 || *output == "" {
			return fmt.Errorf("--output required; unexpected positional arguments are refused")
		}
		b, e := packs.Generate(args[1], *path, *runtimeDir)
		if e != nil {
			return e
		}
		argv, e := packs.PreparedCommand(args[1], *runtimeDir, *path)
		if e != nil {
			return e
		}
		if len(argv) > 0 {
			encoded, _ := json.Marshal(argv)
			b = append([]byte("command: "+string(encoded)+"\n"), b...)
		}
		abs, e := filepath.Abs(*output)
		if e != nil {
			return e
		}
		// Refuse replacement: permission expansion requires a new reviewed policy.
		if e = writeExclusive(abs, b); e != nil {
			return e
		}
		p, _ := packs.Find(args[1])
		metadata := struct {
			Pack         packs.Pack `json:"pack"`
			PolicySHA256 string     `json:"policy_sha256"`
			Evidence     string     `json:"evidence"`
		}{p, packs.Digest(b), "candidate; upstream workflow not verified; local runtime identity not attested"}
		mb, _ := json.MarshalIndent(metadata, "", "  ")
		if e = writeExclusive(abs+".pack.json", mb); e != nil {
			_ = os.Remove(abs)
			return fmt.Errorf("metadata write failed; policy creation rolled back")
		}
		fmt.Fprintln(os.Stderr, "Created candidate policy and metadata. Review grants and test the prepared upstream before use.")
		return nil
	default:
		return fmt.Errorf("unknown packs subcommand")
	}
}

func writeExclusive(path string, b []byte) error {
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	defer f.Close()
	if e = privatefile.Protect(path, false); e != nil {
		_ = os.Remove(path)
		return e
	}
	if _, e = f.Write(b); e != nil {
		_ = os.Remove(path)
		return e
	}
	return f.Sync()
}

func cmdClientConfig(cmd string, args []string) error {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	client := fs.String("client", "", "host adapter")
	scope := fs.String("scope", "project", "configuration scope")
	config := fs.String("config", "", "explicit config")
	server := fs.String("server", "", "existing server name")
	policyPath := fs.String("policy", "", "policy file")
	backend := fs.String("backend", "auto", "sandbox backend")
	wardenBin := fs.String("warden-bin", "", "Warden executable")
	dry := fs.Bool("dry-run", false, "preview")
	yes := fs.Bool("yes", false, "apply")
	output := fs.String("output", "", "export configuration")
	usePolicyCommand := fs.Bool("use-policy-command", false, "use prepared command from policy")
	if e := fs.Parse(args); e != nil {
		return fmt.Errorf("invalid options; use warden %s --help", cmd)
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments; --server must name an existing entry")
	}
	c, e := clientconfig.Get(*client)
	if e != nil {
		return e
	}
	if *client == "claude-desktop" && *scope == "project" {
		*scope = "user"
	}
	path, e := clientconfig.ConfigPath(*client, *scope, *config)
	if e != nil {
		return e
	}
	if cmd == "inventory" {
		entries, e := clientconfig.Inventory(path, c.Format)
		if e != nil {
			return e
		}
		for _, entry := range entries {
			b, _ := json.Marshal(entry)
			fmt.Println(string(b))
		}
		return nil
	}
	if *server == "" {
		return fmt.Errorf("--server is required")
	}
	if cmd == "unwrap" {
		changed, e := clientconfig.Undo(path, *server, *dry || !*yes)
		if e != nil {
			return e
		}
		if !changed {
			fmt.Fprintln(os.Stderr, "No active change to undo.")
		} else if *dry || !*yes {
			fmt.Fprintln(os.Stderr, "Undo preview passed. Use --yes to restore the launcher; unrelated settings stay intact.")
		} else {
			fmt.Fprintln(os.Stderr, "Original launcher restored. Restart your client.")
		}
		return nil
	}
	if *policyPath == "" {
		return fmt.Errorf("--policy is required")
	}
	policyBytes, e := os.ReadFile(*policyPath)
	if e != nil {
		return fmt.Errorf("policy failed validation")
	}
	reviewedHash := clientconfig.Hash(policyBytes)
	p, e := policy.LoadVerified(*policyPath, reviewedHash)
	if e != nil {
		return fmt.Errorf("policy failed validation")
	}
	if *wardenBin == "" {
		*wardenBin, e = os.Executable()
		if e != nil {
			return e
		}
	}
	if !filepath.IsAbs(*wardenBin) {
		return fmt.Errorf("--warden-bin must be absolute")
	}
	info, e := os.Stat(*wardenBin)
	if e != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("Warden executable is unavailable")
	}
	if !validSetupBackend(*backend) {
		return fmt.Errorf("invalid backend")
	}
	var upstream []string
	if *usePolicyCommand {
		if len(p.Command) == 0 {
			return fmt.Errorf("policy has no prepared command")
		}
		upstream = p.Command
	}
	plan, e := clientconfig.PrepareWithCommand(path, c.Format, *server, *policyPath, *wardenBin, *backend, upstream)
	if e != nil {
		return e
	}
	if plan.PolicyHash != reviewedHash {
		return fmt.Errorf("policy changed during setup; no changes applied")
	}
	if plan.Unchanged {
		fmt.Fprintln(os.Stderr, "Already wrapped with the same policy. No changes made.")
		return nil
	}
	storage := clientconfig.StateDirectory(plan.Config)
	for _, grant := range append(append([]string{}, p.Filesystem.Read...), p.Filesystem.Write...) {
		if policy.Within(storage, grant) || policy.Within(grant, storage) {
			return fmt.Errorf("private backup storage overlaps a policy grant; choose a separate WARDEN_SETUP_STATE_DIR or narrower data/runtime grant")
		}
		if real, e := filepath.EvalSymlinks(grant); e == nil && (policy.Within(storage, real) || policy.Within(real, storage)) {
			return fmt.Errorf("resolved policy grant overlaps private backup storage")
		}
	}
	// Avoid command arguments/env/config bodies in previews: they may contain secrets.
	fmt.Fprintf(os.Stderr, "Preview: wrap one stdio launcher for %s. Other fields are preserved.\nGrants: %d read paths, %d write paths, %d hosts, %d environment names.\nPolicy SHA-256: %s\n", c.Name, len(p.Filesystem.Read), len(p.Filesystem.Write), len(p.Network.Allow), len(p.Env.Allow), plan.PolicyHash)
	grants, _ := json.Marshal(struct {
		Read     []string `json:"read"`
		Write    []string `json:"write"`
		Hosts    []string `json:"hosts"`
		EnvNames []string `json:"env_names"`
	}{p.Filesystem.Read, p.Filesystem.Write, p.Network.Allow, p.Env.Allow})
	fmt.Fprintln(os.Stderr, string(grants))
	if *usePolicyCommand {
		fmt.Fprintln(os.Stderr, "Upstream argv will use the reviewed policy command; original argv retained for undo.")
	}
	if *dry {
		fmt.Fprintln(os.Stderr, "Dry run: no files written, no processes started.")
		return nil
	}
	if *output != "" {
		out, e := filepath.Abs(*output)
		if e != nil {
			return e
		}
		if out == plan.Config {
			return fmt.Errorf("export must not overwrite source configuration")
		}
		if e = writeExclusive(out, plan.After); e != nil {
			return e
		}
		fmt.Fprintln(os.Stderr, "Export created with private permissions. Review it before registering it in your host.")
		return nil
	}
	if !*yes {
		fmt.Fprintln(os.Stderr, "Use --yes to apply after readiness checks, or --output to export.")
		return nil
	}
	// Exercise the real sandbox with Warden's own inert probe, never the target
	// server. This catches missing kernel primitives beyond executable detection.
	probePolicy := policy.Policy{Limits: policy.Limits{TimeoutS: 10, MemoryMB: 128}}
	self, e := os.Executable()
	if e != nil {
		return e
	}
	code, e := sandbox.Run([]string{self, "__setup-probe"}, probePolicy, *backend)
	if e != nil || code != 0 {
		return fmt.Errorf("sandbox readiness probe failed; config remains unchanged: %v", e)
	}
	_, e = clientconfig.Apply(plan)
	if e != nil {
		return e
	}
	fmt.Fprintln(os.Stderr, "Applied. Backup and undo record saved in Warden's private user configuration storage.\nRestart your client; verify an allowed task and a blocked task. This connection is configured, not yet workflow-verified.")
	return nil
}

func validSetupBackend(s string) bool {
	for _, b := range []string{"auto", "linux", "seatbelt", "windows", "docker"} {
		if s == b {
			return true
		}
	}
	return false
}
