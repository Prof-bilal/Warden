package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/gateway"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/sandbox"
)

func cmdGateway(args []string) {
	if len(args) == 0 {
		printGatewayUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "init":
		cmdGatewayInit(args[1:])
	case "run":
		cmdGatewayRun(args[1:])
	case "wrap":
		cmdGatewayWrap(args[1:])
	case "list":
		cmdGatewayList(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "warden gateway: unknown subcommand %q\n", args[0])
		printGatewayUsage()
		os.Exit(2)
	}
}

func printGatewayUsage() {
	fmt.Fprintln(os.Stderr, `warden gateway - wrap MCP gateway-registered servers with Warden

Usage:
  warden gateway init --config <file> --policies <dir>
      Generate deny-by-default per-server policies (<name>.yaml) for every
      stdio server in a gateway config. Never overwrites existing files.
      Remote (SSE/HTTP) servers are skipped — they have no local process.

  warden gateway run --config <file> --policies <dir> --server <name>
      [--backend auto|linux|seatbelt|windows|docker] [--approve] [--approve-timeout <dur>] -- [extra args...]
      Run one registered stdio server sandboxed under policies/<name>.yaml.
      Fails closed when the policy is missing or the server is remote.

  warden gateway wrap --config <file> --policies <dir>
      [--warden-bin <path>] [--backend <name>] [--output <file>]
      Print (or write) a copy of the gateway config whose stdio commands are
      prefixed with 'warden run --policy ...', so the gateway itself launches
      every server sandboxed. Point the gateway at the wrapped file.

  warden gateway list --config <file> [--policies <dir>]
      Show registered servers, stdio vs remote, and policy presence.

Supported configs: Claude-style mcpServers JSON (Claude Desktop/Code, Cursor,
VS Code "servers", Docker MCP Gateway clients) and gateway YAML registries
(upstreams: jonfairbanks/mcp-gateway; backends: MikkoParkkola/mcp-gateway).`)
}

func gatewayFlag(args []string, name string) (string, []string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--"+name && i+1 < len(args) {
			return args[i+1], removeArgs(args, i, 2), nil
		}
		if strings.HasPrefix(args[i], "--"+name+"=") {
			return strings.TrimPrefix(args[i], "--"+name+"="), removeArgs(args, i, 1), nil
		}
	}
	return "", args, fmt.Errorf("missing required flag --%s", name)
}

func gatewayOpt(args []string, name string) (string, []string) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--"+name && i+1 < len(args) {
			return args[i+1], removeArgs(args, i, 2)
		}
		if strings.HasPrefix(args[i], "--"+name+"=") {
			return strings.TrimPrefix(args[i], "--"+name+"="), removeArgs(args, i, 1)
		}
	}
	return "", args
}

// removeArgs returns a copy of args with n items at index i dropped. It must
// not alias the input backing array: sequential flag extraction re-slices the
// result, and an in-place append would corrupt the remaining flags.
func removeArgs(args []string, i, n int) []string {
	out := make([]string, 0, len(args)-n)
	out = append(out, args[:i]...)
	return append(out, args[i+n:]...)
}

func cmdGatewayInit(args []string) {
	configPath, rest, err := gatewayFlag(args, "config")
	if err != nil {
		failGateway("init", err)
	}
	policiesDir, rest, err := gatewayFlag(rest, "policies")
	if err != nil {
		failGateway("init", err)
	}
	if len(rest) != 0 {
		failGateway("init", fmt.Errorf("unexpected arguments: %v", rest))
	}
	cfg, err := gateway.Load(configPath)
	if err != nil {
		failGateway("init", err)
	}
	written, skipped, err := gateway.WritePolicies(cfg, policiesDir)
	if err != nil {
		failGateway("init", err)
	}
	for _, n := range written {
		fmt.Fprintf(os.Stderr, "warden gateway init: wrote %s\n", gateway.PolicyPath(policiesDir, n))
	}
	for _, s := range skipped {
		fmt.Fprintf(os.Stderr, "warden gateway init: skip %s\n", s)
	}
	if len(written) == 0 {
		fmt.Fprintln(os.Stderr, "warden gateway init: nothing to sandbox (all entries remote or already have policies)")
		return
	}
	fmt.Fprintf(os.Stderr, "warden gateway init: next, widen filesystem/network with `warden trace` + edit, then `warden gateway run --config %s --policies %s --server <name>`\n", configPath, policiesDir)
}

func cmdGatewayList(args []string) {
	configPath, rest, err := gatewayFlag(args, "config")
	if err != nil {
		failGateway("list", err)
	}
	policiesDir, rest := gatewayOpt(rest, "policies")
	if len(rest) != 0 {
		failGateway("list", fmt.Errorf("unexpected arguments: %v", rest))
	}
	cfg, err := gateway.Load(configPath)
	if err != nil {
		failGateway("list", err)
	}
	for _, s := range cfg.Servers {
		kind := "stdio"
		detail := strings.Join(s.Command, " ")
		if s.Remote {
			kind = "remote"
			detail = s.Reason
		}
		status := ""
		if policiesDir != "" && !s.Remote {
			if _, err := os.Stat(gateway.PolicyPath(policiesDir, s.Name)); err == nil {
				status = " [policy ok]"
			} else {
				status = " [no policy]"
			}
		}
		fmt.Fprintf(os.Stdout, "%s\t%s\t%s%s\n", s.Name, kind, detail, status)
	}
}

func cmdGatewayRun(args []string) {
	configPath, rest, err := gatewayFlag(args, "config")
	if err != nil {
		failGateway("run", err)
	}
	policiesDir, rest, err := gatewayFlag(rest, "policies")
	if err != nil {
		failGateway("run", err)
	}
	serverName, rest, err := gatewayFlag(rest, "server")
	if err != nil {
		failGateway("run", err)
	}
	backend, rest := gatewayOpt(rest, "backend")
	approveTimeoutStr, rest := gatewayOpt(rest, "approve-timeout")
	approveEnabled, rest := gatewayBoolOpt(rest, "approve")
	var approveTimeout time.Duration
	if approveTimeoutStr != "" || approveEnabled {
		if approveTimeoutStr != "" && !approveEnabled {
			failGateway("run", fmt.Errorf("--approve-timeout requires --approve"))
		}
		if approveTimeoutStr != "" {
			var err error
			approveTimeout, err = time.ParseDuration(approveTimeoutStr)
			if err != nil || approveTimeout < 0 {
				failGateway("run", fmt.Errorf("--approve-timeout %q: want a non-negative duration like 30s or 2m", approveTimeoutStr))
			}
		}
		if err := approve.CheckTTY(); err != nil {
			failGateway("run", err)
		}
	}
	var extra []string
	for i, a := range rest {
		if a == "--" {
			extra = rest[i+1:]
			rest = rest[:i]
			break
		}
	}
	if len(rest) != 0 {
		failGateway("run", fmt.Errorf("unexpected arguments: %v (pass extra server args after --)", rest))
	}

	cfg, err := gateway.Load(configPath)
	if err != nil {
		failGateway("run", err)
	}
	entry, err := cfg.Find(serverName)
	if err != nil {
		failGateway("run", err)
	}
	if entry.Remote {
		failGateway("run", fmt.Errorf("server %q is remote (%s); no local process to sandbox", entry.Name, entry.URL))
	}

	policyPath := gateway.PolicyPath(policiesDir, entry.Name)
	if approveEnabled {
		cmdGatewayRunApproval(configPath, policiesDir, entry, backend, extra, policyPath, approveTimeout)
		return // unreachable: cmdGatewayRunApproval always exits
	}
	p, err := policy.Load(policyPath)
	if err != nil {
		if os.IsNotExist(err) {
			failGateway("run", fmt.Errorf("no policy %q; run `warden gateway init --config %s --policies %s` first", policyPath, configPath, policiesDir))
		}
		failGateway("run", err)
	}
	for _, advisory := range p.UnimplementedAdvisories() {
		fmt.Fprintf(os.Stderr, "warden warning: %s\n", advisory)
	}

	cmd := append(append([]string{}, entry.Command...), extra...)
	if len(cmd) == 0 {
		failGateway("run", fmt.Errorf("server %q has no command", entry.Name))
	}
	// Same PATH resolution as `warden run`: gateway entries commonly name a
	// bare launcher (npx/uvx). StarterPolicy resolves at init time; resolve
	// again here for hand-edited configs. Still fail-closed off PATH.
	resolved, err := policy.ResolveExecutable(cmd)
	if err != nil {
		failGateway("run", err)
	}
	cmd = resolved
	if _, err := os.Stat(cmd[0]); err != nil {
		failGateway("run", fmt.Errorf("executable %q: %v", cmd[0], err))
	}
	if len(p.Command) > 0 && !sameCommand(p.Command, entry.Command) {
		fmt.Fprintf(os.Stderr, "warden warning: policy command %q differs from gateway entry %q; running the gateway entry\n", p.Command, entry.Command)
	}

	// Inject gateway env VALUES for allowlisted names (with ${VAR} expansion),
	// so `gateway run` behaves like the gateway spawning the server itself.
	// Values never touch the policy file.
	if err := injectGatewayEnv(entry, p); err != nil {
		failGateway("run", err)
	}

	exitCode, err := sandbox.Run(cmd, p, backend)
	if err != nil {
		failGateway("run", err)
	}
	os.Exit(exitCode)
}

// cmdGatewayRunApproval runs one gateway server with interactive approval,
// respawning when a filesystem approval requires a restart. The policy is
// reloaded every iteration so saved grants take effect. It always exits.
func cmdGatewayRunApproval(configPath, policiesDir string, entry gateway.ServerEntry, backend string, extra []string, policyPath string, timeout time.Duration) {
	cfg := &approve.Config{Enabled: true, PolicyPath: policyPath, Timeout: timeout}
	if err := cfg.Validate(); err != nil {
		failGateway("run", err)
	}
	for i := 0; i < maxApprovalRestarts; i++ {
		p, err := policy.Load(policyPath)
		if err != nil {
			if os.IsNotExist(err) {
				failGateway("run", fmt.Errorf("no policy %q; run `warden gateway init --config %s --policies %s` first", policyPath, configPath, policiesDir))
			}
			failGateway("run", err)
		}
		for _, advisory := range p.UnimplementedAdvisories() {
			fmt.Fprintf(os.Stderr, "warden warning: %s\n", advisory)
		}
		cmd := append(append([]string{}, entry.Command...), extra...)
		if err := checkGatewayCommand(entry, cmd); err != nil {
			failGateway("run", err)
		}
		if len(p.Command) > 0 && !sameCommand(p.Command, entry.Command) {
			fmt.Fprintf(os.Stderr, "warden warning: policy command %q differs from gateway entry %q; running the gateway entry\n", p.Command, entry.Command)
		}
		if err := injectGatewayEnv(entry, p); err != nil {
			failGateway("run", err)
		}
		exitCode, err := sandbox.RunWithApproval(cmd, p, backend, cfg)
		if errors.Is(err, approve.ErrRestartRequested) {
			fmt.Fprintf(os.Stderr, "warden gateway run: restarting with the approved policy (%d/%d)\n", i+1, maxApprovalRestarts)
			continue
		}
		if err != nil {
			failGateway("run", err)
		}
		os.Exit(exitCode)
	}
	failGateway("run", fmt.Errorf("too many approval restarts (%d); inspect %s for runaway grants", maxApprovalRestarts, policyPath))
}

func checkGatewayCommand(entry gateway.ServerEntry, cmd []string) error {
	if len(cmd) == 0 {
		return fmt.Errorf("server %q has no command", entry.Name)
	}
	return checkSandboxCommand(cmd)
}

// gatewayBoolOpt extracts a bare boolean flag (present = true).
func gatewayBoolOpt(args []string, name string) (bool, []string) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--"+name {
			return true, removeArgs(args, i, 1)
		}
	}
	return false, args
}

func cmdGatewayWrap(args []string) {
	configPath, rest, err := gatewayFlag(args, "config")
	if err != nil {
		failGateway("wrap", err)
	}
	policiesDir, rest, err := gatewayFlag(rest, "policies")
	if err != nil {
		failGateway("wrap", err)
	}
	wardenBin, rest := gatewayOpt(rest, "warden-bin")
	if wardenBin == "" {
		if ex, err := os.Executable(); err == nil {
			wardenBin = ex
		} else {
			failGateway("wrap", fmt.Errorf("cannot detect warden binary path; pass --warden-bin explicitly"))
		}
	}
	backend, rest := gatewayOpt(rest, "backend")
	output, rest := gatewayOpt(rest, "output")
	if len(rest) != 0 {
		failGateway("wrap", fmt.Errorf("unexpected arguments: %v", rest))
	}
	cfg, err := gateway.Load(configPath)
	if err != nil {
		failGateway("wrap", err)
	}
	// Fail closed: every stdio server must already have a policy, otherwise
	// the wrapped gateway would reference a nonexistent file.
	var missing []string
	for _, s := range cfg.Stdio() {
		if _, err := os.Stat(gateway.PolicyPath(policiesDir, s.Name)); err != nil {
			missing = append(missing, s.Name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		failGateway("wrap", fmt.Errorf("missing policies for %s; run `warden gateway init --config %s --policies %s` first", strings.Join(missing, ", "), configPath, policiesDir))
	}
	out, err := gateway.Wrap(cfg, policiesDir, wardenBin, backend)
	if err != nil {
		failGateway("wrap", err)
	}
	if output == "" {
		if _, err := os.Stdout.Write(out); err != nil {
			failGateway("wrap", err)
		}
		return
	}
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			failGateway("wrap", fmt.Errorf("%q already exists; refusing to overwrite it", output))
		}
		failGateway("wrap", fmt.Errorf("create %q: %v", output, err))
	}
	if _, err := f.Write(out); err != nil {
		f.Close()
		failGateway("wrap", fmt.Errorf("write %q: %v", output, err))
	}
	if err := f.Close(); err != nil {
		failGateway("wrap", fmt.Errorf("close %q: %v", output, err))
	}
	fmt.Fprintf(os.Stderr, "warden gateway wrap: wrote wrapped config to %s; point the gateway at it\n", output)
}

func failGateway(sub string, err error) {
	fmt.Fprintf(os.Stderr, "warden gateway %s: %v\n", sub, err)
	os.Exit(2)
}

func sameCommand(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// injectGatewayEnv exports gateway entry env values for names in the policy
// allowlist, expanding ${VAR} (required) and ${VAR:-default} templates. Keys
// not in the policy allowlist are ignored (deny by default). Pre-existing
// process env wins over the gateway default, so local exports take precedence.
func injectGatewayEnv(entry gateway.ServerEntry, p policy.Policy) error {
	allowed := make(map[string]bool, len(p.Env.Allow))
	for _, n := range p.EnvAllowlist() {
		allowed[n] = true
	}
	for k, raw := range entry.Env {
		if !allowed[k] {
			continue
		}
		if _, exists := os.LookupEnv(k); exists {
			continue
		}
		val, err := expandEnvTemplate(raw)
		if err != nil {
			return fmt.Errorf("server %q env %s: %w", entry.Name, k, err)
		}
		if err := os.Setenv(k, val); err != nil {
			return fmt.Errorf("export %s: %w", k, err)
		}
	}
	return nil
}

// expandEnvTemplate expands ${NAME} and ${NAME:-default} in v. A required var
// that is unset is an error (fail closed, never silently empty a credential).
func expandEnvTemplate(v string) (string, error) {
	var sb strings.Builder
	for {
		start := strings.Index(v, "${")
		if start < 0 {
			sb.WriteString(v)
			break
		}
		sb.WriteString(v[:start])
		rest := v[start+2:]
		end := strings.Index(rest, "}")
		if end < 0 {
			return "", fmt.Errorf("unterminated ${ in %q", v)
		}
		expr := rest[:end]
		v = rest[end+1:]
		name, def, hasDef := strings.Cut(expr, ":-")
		if strings.TrimSpace(name) == "" || strings.ContainsAny(name, " \t\n=") {
			return "", fmt.Errorf("invalid variable reference ${%s}", expr)
		}
		val, ok := os.LookupEnv(name)
		if (!ok || val == "") && hasDef {
			val = def
		} else if !ok {
			return "", fmt.Errorf("environment variable %s is not set (referenced as ${%s}); export it or use ${%s:-default}", name, expr, name)
		}
		sb.WriteString(val)
	}
	return sb.String(), nil
}
