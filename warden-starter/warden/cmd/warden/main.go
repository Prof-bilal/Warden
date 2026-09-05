// Command warden is the CLI entry point for the Warden sandbox runtime.
//
// Subcommands:
//   - `warden run --policy <file> -- <command...>` runs a server under a
//     policy, sandboxed by the current OS backend (Linux: bubblewrap).
//   - `warden trace`, `warden init`, `warden logs` are stubs for M2/M3.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/sandbox/linux"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		cmdRun(os.Args[2:])
	case "trace":
		cmdTrace(os.Args[2:])
	case "init":
		cmdInit(os.Args[2:])
	case "logs":
		cmdLogs(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `warden - a sandbox runtime for MCP servers

Usage:
  warden run --policy <file> -- <command...>   Run a server under a policy
  warden trace -- <command...>                 Run unsandboxed, log all access attempts (M3)
  warden init                                  Generate a starter policy from a trace log (M3)
  warden logs                                  View the audit log (M2)

See ROADMAP.md. M1 (Linux filesystem sandboxing) is implemented; network
enforcement and audit logging arrive in M2/M3.`)
}

func cmdRun(args []string) {
	policyPath, cmdTail, err := parseRunArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		printUsage()
		os.Exit(2)
	}

	p, err := policy.Load(policyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		os.Exit(2)
	}

	for _, advisory := range p.UnimplementedAdvisories() {
		fmt.Fprintf(os.Stderr, "warden warning: %s\n", advisory)
	}

	cmd, err := p.ResolveCommand(cmdTail)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		os.Exit(2)
	}

	// The sandbox needs an absolute path for the executable: bwrap binds
	// the parent dir, and a bare command name would resolve nowhere.
	if !filepath.IsAbs(cmd[0]) {
		fmt.Fprintf(os.Stderr, "warden run: command %q is not an absolute path; sandbox requires a full path to the executable\n", cmd[0])
		os.Exit(2)
	}
	if _, err := os.Stat(cmd[0]); err != nil {
		fmt.Fprintf(os.Stderr, "warden run: executable %q: %v\n", cmd[0], err)
		os.Exit(2)
	}

	exitCode, err := linux.Run(cmd, p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		os.Exit(1)
	}
	os.Exit(exitCode)
}

// parseRunArgs splits `--policy <file> <command...>` into the policy path
// and the command tail. A `--` separator is optional but supported: it lets
// the command start with a flag named like a warden option.
func parseRunArgs(args []string) (policyPath string, cmd []string, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--policy" || arg == "-policy":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("flag %s requires a value", arg)
			}
			if policyPath != "" {
				return "", nil, fmt.Errorf("--policy specified twice")
			}
			policyPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "--policy="):
			if policyPath != "" {
				return "", nil, fmt.Errorf("--policy specified twice")
			}
			policyPath = strings.TrimPrefix(arg, "--policy=")
		case arg == "--":
			cmd = args[i+1:]
			if policyPath == "" {
				return "", nil, fmt.Errorf("missing required flag --policy")
			}
			return policyPath, cmd, nil
		case strings.HasPrefix(arg, "-"):
			return "", nil, fmt.Errorf("unknown flag %q (want --policy)", arg)
		default:
			// First non-flag argument: everything from here is the command.
			cmd = args[i:]
			if policyPath == "" {
				return "", nil, fmt.Errorf("missing required flag --policy")
			}
			return policyPath, cmd, nil
		}
	}
	if policyPath == "" {
		return "", nil, fmt.Errorf("missing required flag --policy")
	}
	return policyPath, nil, nil
}

func cmdTrace(args []string) {
	// TODO(M3): run the given command unsandboxed but instrumented,
	// logging file/network access attempts via internal/audit.
	fmt.Println("warden trace: not yet implemented — see ROADMAP.md milestone M3")
}

func cmdInit(args []string) {
	// TODO(M3): read a trace log and emit a starter policy.yaml.
	fmt.Println("warden init: not yet implemented — see ROADMAP.md milestone M3")
}

func cmdLogs(args []string) {
	// TODO(M2): read and pretty-print the structured audit log.
	fmt.Println("warden logs: not yet implemented — see ROADMAP.md milestone M2")
}
