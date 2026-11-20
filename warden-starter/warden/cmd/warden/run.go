package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/sandbox"
	"github.com/warden-sandbox/warden/internal/sandbox/sandboxerr"
	"github.com/warden-sandbox/warden/internal/ui"
	"gopkg.in/yaml.v3"
)

func cmdRun(args []string) {
	flags, cmdTail := splitRunCommand(args)
	expectedDigest, flags, digestErr := parsePolicyDigest(flags)
	if digestErr != nil {
		fmt.Fprintln(os.Stderr, "warden run:", digestErr)
		os.Exit(2)
	}
	approveEnabled, approveTimeout, rest, err := parseApproveFlags(flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		printRunHelp()
		os.Exit(2)
	}
	policyPath, backend, _, dryRun, err := parseRunArgs(rest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		printRunHelp()
		os.Exit(2)
	}

	var provenance string
	policyPath, err = runPolicyPath(policyPath, len(cmdTail) > 0)
	if errors.Is(err, errNoRunPolicy) && expectedDigest == "" && !approveEnabled && !dryRun {
		provenance = "pack"
		policyPath, expectedDigest, err = resolveRunPack(cmdTail, backend)
		if errors.Is(err, errNoRunPolicy) {
			provenance = "wizard"
			policyPath, expectedDigest, err = resolveRunWizard(cmdTail, backend)
		}
		if err == nil {
			cmdTail = nil // Run the command in the generated, pinned policy.
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		os.Exit(2)
	}

	if approveEnabled {
		if dryRun {
			fmt.Fprintln(os.Stderr, "warden run: --dry-run cannot be combined with --approve")
			os.Exit(2)
		}
		if expectedDigest != "" {
			fmt.Fprintln(os.Stderr, "warden run: digest-pinned policies require explicit review instead of automatic approval mutations")
			os.Exit(2)
		}
		cmdRunWithApproval(policyPath, backend, cmdTail, approveTimeout)
		return // unreachable: cmdRunWithApproval always exits
	}

	p, digest, provenance, err := resolvePolicy(policySource{path: policyPath, pin: expectedDigest, provenance: provenance})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		os.Exit(2)
	}

	for _, advisory := range p.Advisories() {
		fmt.Fprintf(os.Stderr, "warden warning: %s\n", advisory)
	}

	cmd, err := p.ResolveCommand(cmdTail)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		os.Exit(2)
	}

	// Most public MCP servers launch via a bare launcher (npx/uvx/node on
	// PATH). Resolve it to an absolute path like the gateway backend does,
	// so the sandbox can bind-mount the executable's parent directory.
	// Still fail-closed when the name is not on PATH.
	cmd, err = policy.ResolveExecutable(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		os.Exit(2)
	}

	if err := checkSandboxCommand(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		os.Exit(2)
	}

	if dryRun {
		if err := printRunPlan(policyPath, digest, provenance, backend, p, cmd); err != nil {
			fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
			os.Exit(2)
		}
		return
	}

	displayPath := policyPath
	if provenance != "file" {
		displayPath = fmt.Sprintf("%s (%s)", policyPath, provenance)
	}
	maybePrintRunSummary(displayPath, backend, p, cmd)

	exitCode, err := sandbox.Run(cmd, p, backend)
	if err != nil {
		if handled, code := offerElevation(err); handled {
			os.Exit(code)
		}
		var ref sandboxerr.RefuseToRun
		if errors.As(err, &ref) {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		}
		os.Exit(1)
	}
	os.Exit(exitCode)
}

func printRunPlan(path, digest, provenance, backend string, p policy.Policy, cmd []string) error {
	if backend == "" {
		backend = "auto"
	}
	selected, err := sandbox.Resolve(backend)
	if err != nil {
		return err
	}
	p = p.WithActiveGrants(time.Now())
	p.Command = cmd
	plan := struct {
		Platform      string        `yaml:"platform"`
		Backend       string        `yaml:"backend"`
		Source        string        `yaml:"policy_path"`
		Provenance    string        `yaml:"policy_source"`
		Digest        string        `yaml:"policy_sha256"`
		GrantDeadline time.Time     `yaml:"grant_deadline,omitempty"`
		Policy        policy.Policy `yaml:"policy"`
	}{
		Platform:      runtime.GOOS + "/" + runtime.GOARCH,
		Backend:       fmt.Sprint(selected),
		Source:        path,
		Provenance:    provenance,
		Digest:        digest,
		GrantDeadline: p.Limits.GrantDeadline,
		Policy:        p,
	}
	data, err := yaml.Marshal(plan)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}

// splitRunCommand keeps Warden's flag parsers out of the launcher's arguments.
func splitRunCommand(args []string) (flags, command []string) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--":
			return args[:i], args[i+1:]
		case "--policy", "-policy", "--backend", "-backend", "--policy-sha256", "--approve-timeout":
			i++
		default:
			if !strings.HasPrefix(args[i], "-") {
				return args[:i], args[i:]
			}
		}
	}
	return args, nil
}

// parseRunArgs splits run flags into the policy path, optional backend, and
// command tail. A `--` separator is optional but supported: it lets the
// command start with a flag named like a warden option.
func parseRunArgs(args []string) (policyPath, backend string, cmd []string, dryRun bool, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--policy" || arg == "-policy":
			if i+1 >= len(args) || args[i+1] == "" {
				return "", "", nil, false, fmt.Errorf("flag %s requires a value", arg)
			}
			if policyPath != "" {
				return "", "", nil, false, fmt.Errorf("--policy specified twice")
			}
			policyPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "--policy="):
			if policyPath != "" {
				return "", "", nil, false, fmt.Errorf("--policy specified twice")
			}
			policyPath = strings.TrimPrefix(arg, "--policy=")
			if policyPath == "" {
				return "", "", nil, false, fmt.Errorf("flag --policy requires a value")
			}
		case arg == "--backend" || arg == "-backend":
			if i+1 >= len(args) {
				return "", "", nil, false, fmt.Errorf("flag %s requires a value", arg)
			}
			if backend != "" {
				return "", "", nil, false, fmt.Errorf("--backend specified twice")
			}
			backend = args[i+1]
			i++
		case strings.HasPrefix(arg, "--backend="):
			if backend != "" {
				return "", "", nil, false, fmt.Errorf("--backend specified twice")
			}
			backend = strings.TrimPrefix(arg, "--backend=")
		case arg == "--dry-run":
			dryRun = true
		case arg == "--":
			return policyPath, backend, args[i+1:], dryRun, nil
		case strings.HasPrefix(arg, "-"):
			return "", "", nil, false, fmt.Errorf("unknown flag %q (want --policy, --backend, or --dry-run)", arg)
		default:
			return policyPath, backend, args[i:], dryRun, nil
		}
	}
	return policyPath, backend, nil, dryRun, nil
}

func printRunHelp() {
	title := "WARDEN RUN"
	if ui.ColorEnabled() {
		title = ui.Bold(ui.Cyan(title))
	}
	fmt.Fprintln(os.Stderr, title)
	fmt.Fprintln(os.Stderr, "Run an MCP server inside the Warden sandbox.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Usage:"))
	fmt.Fprintln(os.Stderr, "  warden run [options] [--] <command...>")
	fmt.Fprintln(os.Stderr, "  warden run --policy <file> [options]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Options:"))
	fmt.Fprintln(os.Stderr, "  --policy <file>            Security policy to enforce (defaults to ./warden.yaml)")
	fmt.Fprintln(os.Stderr, "  --policy-sha256 <digest>   Refuse a policy changed since review (used by wrap)")
	fmt.Fprintln(os.Stderr, "  --backend <name>           Backend: auto (default), linux, seatbelt, windows, docker")
	fmt.Fprintln(os.Stderr, "  --dry-run                  Print the plan for an existing policy without running the command")
	fmt.Fprintln(os.Stderr, "  --approve                  Prompt on first out-of-policy access")
	fmt.Fprintln(os.Stderr, "  --approve-timeout <dur>    Per-prompt timeout (e.g. 30s, 2m); requires --approve")
	fmt.Fprintln(os.Stderr, "  --help                     Show this help")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Examples:"))
	fmt.Fprintln(os.Stderr, "  warden run npx @modelcontextprotocol/server-filesystem")
	fmt.Fprintln(os.Stderr, "  warden run --policy policy.yaml -- /usr/bin/node server.js")
	fmt.Fprintln(os.Stderr, "  warden run --policy policy.yaml --backend docker -- python app.py")
	fmt.Fprintln(os.Stderr, "  warden run --policy policy.yaml --approve -- npx @modelcontextprotocol/server")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Security:"))
	fmt.Fprintln(os.Stderr, "  The server runs with only the permissions explicitly")
	fmt.Fprintln(os.Stderr, "  granted by the policy. If the sandbox cannot be")
	fmt.Fprintln(os.Stderr, "  initialized, Warden refuses to run the server.")
}

func runSandboxWithEnv(args []string, p policy.Policy, backend string, vars map[string]string) (int, error) {
	restore := map[string]*string{}
	defer func() {
		for key, value := range restore {
			if value == nil {
				_ = os.Unsetenv(key)
			} else {
				_ = os.Setenv(key, *value)
			}
		}
	}()
	for key, value := range vars {
		if saved, ok := os.LookupEnv(key); ok {
			restore[key] = &saved
		} else {
			restore[key] = nil
		}
		if err := os.Setenv(key, value); err != nil {
			return 0, err
		}
		p.Env.Allow = append(p.Env.Allow, key)
	}
	p.Normalize()
	if e := p.Validate(); e != nil {
		return 0, e
	}
	return sandbox.Run(args, p, backend)
}
