// Command warden is the CLI entry point for the Warden sandbox runtime.
//
// Subcommands:
//   - `warden run --policy <file> [--backend auto|linux|seatbelt|docker] --
//     <command...>` runs a server under a policy. Backend auto-detection
//     prefers the OS-native sandbox (bwrap / sandbox-exec) and falls back to
//     Docker only when the native primitive is missing.
//   - `warden trace`, `warden init`, and `warden logs` are usability tools.
//   - `warden version` (`--version`) prints the stamped build version.
//   - `warden doctor` checks sandbox readiness.
//   - `warden` with no arguments shows the branded header and usage.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/container"
	"github.com/warden-sandbox/warden/internal/mcpproxy"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/proxy"
	"github.com/warden-sandbox/warden/internal/sandbox"
	"github.com/warden-sandbox/warden/internal/sandbox/sandboxerr"
	"github.com/warden-sandbox/warden/internal/selfupdate"
	"github.com/warden-sandbox/warden/internal/ui"
	"github.com/warden-sandbox/warden/internal/version"
)

// allCommands lists every user-facing command name for help discovery and
// suggestion matching. Internal commands (like __proxy-bridge) are excluded.
var allCommands = []string{"run", "trace", "init", "logs", "doctor", "gateway", "proxy", "k8s", "version", "update", "help"}

func main() {
	if len(os.Args) < 2 {
		handleBareInvocation()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "__proxy-bridge":
		cmdProxyBridge(os.Args[2:])
	case "run":
		if hasHelpFlag(os.Args[2:]) {
			printRunHelp()
			os.Exit(0)
		}
		cmdRun(os.Args[2:])
	case "trace":
		if hasHelpFlag(os.Args[2:]) {
			printTraceHelp()
			os.Exit(0)
		}
		cmdTrace(os.Args[2:])
	case "init":
		if hasHelpFlag(os.Args[2:]) {
			printInitHelp()
			os.Exit(0)
		}
		cmdInit(os.Args[2:])
	case "logs":
		if hasHelpFlag(os.Args[2:]) {
			printLogsHelp()
			os.Exit(0)
		}
		cmdLogs(os.Args[2:])
	case "gateway":
		if hasHelpFlag(os.Args[2:]) {
			printGatewayUsage()
			os.Exit(0)
		}
		cmdGateway(os.Args[2:])
	case "proxy":
		if hasHelpFlag(os.Args[2:]) {
			printProxyHelp()
			os.Exit(0)
		}
		cmdProxy(os.Args[2:])
	case "k8s":
		if hasHelpFlag(os.Args[2:]) {
			printK8sHelp()
			os.Exit(0)
		}
		cmdK8s(os.Args[2:])
	case "doctor":
		if hasHelpFlag(os.Args[2:]) {
			printDoctorHelp()
			os.Exit(0)
		}
		cmdDoctor(os.Args[2:])
	case "version", "--version", "-v", "-version":
		if hasHelpFlag(os.Args[2:]) {
			printVersionHelp()
			os.Exit(0)
		}
		cmdVersion()
	case "update":
		if hasHelpFlag(os.Args[2:]) {
			printUpdateHelp()
			os.Exit(0)
		}
		cmdUpdate(os.Args[2:])
	case "help":
		if len(os.Args) > 2 {
			printCommandHelp(os.Args[2])
		} else {
			printUsage()
		}
		os.Exit(0)
	case "--help", "-h":
		printUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "\u2717 Unknown command: %s\n\n", os.Args[1])
		if suggestion := suggestCommand(os.Args[1]); suggestion != "" {
			fmt.Fprintf(os.Stderr, "Did you mean:\n\n  warden %s\n\n", suggestion)
		}
		fmt.Fprintf(os.Stderr, "Run:\n\n  warden help\n\nto see available commands.\n")
		os.Exit(1)
	}
}

func hasHelpFlag(args []string) bool {
	for _, a := range args {
		if a == "--help" || a == "-h" || a == "help" {
			return true
		}
	}
	return false
}

func handleBareInvocation() {
	// First-run welcome takes precedence on first meaningful invocation.
	if shown := ui.MaybeShowWelcome(os.Stderr); shown {
		fmt.Fprintln(os.Stderr, "")
		printUsage()
		return
	}
	// Otherwise show branded header + usage.
	// In a TTY we show the large banner; in non-TTY we rely on printUsage
	// header (which already prints WARDEN + tagline) to avoid duplication.
	if ui.IsTerminalWriter(os.Stderr) && !ui.IsCI() {
		_, _ = ui.PrintBanner(os.Stderr)
		fmt.Fprintln(os.Stderr, "")
	}
	printUsage()
}

func printUsage() {
	title := "WARDEN"
	if ui.ColorEnabled() {
		title = ui.Bold(ui.Cyan(title))
	}
	sub := "Secure execution for MCP servers."
	if ui.ColorEnabled() {
		sub = ui.Dim(sub)
	}
	fmt.Fprintln(os.Stderr, title)
	fmt.Fprintln(os.Stderr, sub)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Usage:"))
	fmt.Fprintln(os.Stderr, "  warden <command> [options]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Commands:"))
	fmt.Fprintln(os.Stderr, "  init       Create a security policy")
	fmt.Fprintln(os.Stderr, "  run        Run an MCP server in the sandbox")
	fmt.Fprintln(os.Stderr, "  trace      Record access attempts for policy generation")
	fmt.Fprintln(os.Stderr, "  logs       Inspect the audit log")
	fmt.Fprintln(os.Stderr, "  doctor     Check sandbox readiness")
	fmt.Fprintln(os.Stderr, "  gateway    Wrap gateway-registered servers")
	fmt.Fprintln(os.Stderr, "  proxy      Run MCP client proxy with filtering")
	fmt.Fprintln(os.Stderr, "  k8s        Generate container/K8s manifests from policy")
	fmt.Fprintln(os.Stderr, "  version    Show version")
	fmt.Fprintln(os.Stderr, "  update     Update warden to the latest release")
	fmt.Fprintln(os.Stderr, "  help       Show help for a command")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Get started:"))
	fmt.Fprintln(os.Stderr, "  warden init")
	fmt.Fprintln(os.Stderr, "  warden run --policy policy.yaml -- <server>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Learn more:"))
	fmt.Fprintln(os.Stderr, "  warden help run")
	fmt.Fprintln(os.Stderr, "  warden help gateway")
	fmt.Fprintln(os.Stderr, "")
	sec := "Security:"
	if ui.ColorEnabled() {
		sec = ui.Cyan(sec)
	}
	fmt.Fprintln(os.Stderr, sec)
	fmt.Fprintln(os.Stderr, "  Warden fails closed when sandbox enforcement")
	fmt.Fprintln(os.Stderr, "  is unavailable. It never runs unsandboxed.")
	fmt.Fprintln(os.Stderr, "  The server gets only the permissions your policy")
	fmt.Fprintln(os.Stderr, "  explicitly grants.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Dim("Docs: https://github.com/Prof-bilal/Warden"))
}

// printCommandHelp dispatches to the right help function for "warden help <cmd>".
func printCommandHelp(cmd string) {
	switch cmd {
	case "run":
		printRunHelp()
	case "trace":
		printTraceHelp()
	case "init":
		printInitHelp()
	case "logs":
		printLogsHelp()
	case "doctor":
		printDoctorHelp()
	case "version":
		printVersionHelp()
	case "update":
		printUpdateHelp()
	case "gateway":
		printGatewayUsage()
	case "proxy":
		printProxyHelp()
	case "k8s":
		printK8sHelp()
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "\u2717 No help available for: %s\n\n", cmd)
		fmt.Fprintf(os.Stderr, "Run:\n\n  warden help\n\nto see available commands.\n")
		os.Exit(1)
	}
}

// suggestCommand returns the closest command name for a typo, or "" if none
// is close enough. Uses simple Levenshtein-distance heuristic.
func suggestCommand(input string) string {
	best := ""
	bestDist := len(input) / 2 // max distance to consider a match
	if bestDist < 1 {
		bestDist = 1
	}
	for _, cmd := range allCommands {
		d := levenshtein(input, cmd)
		if d > 0 && d <= bestDist {
			bestDist = d
			best = cmd
		}
	}
	return best
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev = curr
	}
	return prev[lb]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
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
	fmt.Fprintln(os.Stderr, "  warden run --policy <file> [options] -- <command...>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Options:"))
	fmt.Fprintln(os.Stderr, "  --policy <file>            Security policy to enforce (required)")
	fmt.Fprintln(os.Stderr, "  --backend <name>           Backend: auto (default), linux, seatbelt, windows, docker")
	fmt.Fprintln(os.Stderr, "  --approve                  Prompt on first out-of-policy access")
	fmt.Fprintln(os.Stderr, "  --approve-timeout <dur>    Per-prompt timeout (e.g. 30s, 2m); requires --approve")
	fmt.Fprintln(os.Stderr, "  --help                     Show this help")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Examples:"))
	fmt.Fprintln(os.Stderr, "  warden run --policy policy.yaml -- /usr/bin/node server.js")
	fmt.Fprintln(os.Stderr, "  warden run --policy policy.yaml --backend docker -- python app.py")
	fmt.Fprintln(os.Stderr, "  warden run --policy policy.yaml --approve -- npx @modelcontextprotocol/server")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Security:"))
	fmt.Fprintln(os.Stderr, "  The server runs with only the permissions explicitly")
	fmt.Fprintln(os.Stderr, "  granted by the policy. If the sandbox cannot be")
	fmt.Fprintln(os.Stderr, "  initialized, Warden refuses to run the server.")
}

func printTraceHelp() {
	title := "WARDEN TRACE"
	if ui.ColorEnabled() {
		title = ui.Bold(ui.Cyan(title))
	}
	fmt.Fprintln(os.Stderr, title)
	fmt.Fprintln(os.Stderr, "Run unsandboxed and record access attempts for policy generation.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Usage:"))
	fmt.Fprintln(os.Stderr, "  warden trace -- <command...>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Examples:"))
	fmt.Fprintln(os.Stderr, "  warden trace -- /usr/bin/node server.js")
	fmt.Fprintln(os.Stderr, "  warden trace -- npx @modelcontextprotocol/server")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Next:"))
	fmt.Fprintln(os.Stderr, "  warden init          Generate a policy from the trace")
	fmt.Fprintln(os.Stderr, "  warden run --policy policy.yaml -- <server>")
}

func printInitHelp() {
	title := "WARDEN INIT"
	if ui.ColorEnabled() {
		title = ui.Bold(ui.Cyan(title))
	}
	fmt.Fprintln(os.Stderr, title)
	fmt.Fprintln(os.Stderr, "Create a security policy from a recorded trace.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Usage:"))
	fmt.Fprintln(os.Stderr, "  warden init [options]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Options:"))
	fmt.Fprintln(os.Stderr, "  --log <file>       Audit/trace log to read (default: latest trace)")
	fmt.Fprintln(os.Stderr, "  --output <file>    Output policy file (default: policy.yaml)")
	fmt.Fprintln(os.Stderr, "  -- <command...>    Command to record in the policy")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("What the policy controls:"))
	fmt.Fprintln(os.Stderr, "  Filesystem     explicit paths only")
	fmt.Fprintln(os.Stderr, "  Network        explicit hosts only")
	fmt.Fprintln(os.Stderr, "  Environment    explicit variables only")
	fmt.Fprintln(os.Stderr, "  Resources      explicit limits")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Examples:"))
	fmt.Fprintln(os.Stderr, "  warden init")
	fmt.Fprintln(os.Stderr, "  warden init --log trace.jsonl --output starter.yaml")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Next:"))
	fmt.Fprintln(os.Stderr, "  warden run --policy policy.yaml -- <server>")
}

func printLogsHelp() {
	title := "WARDEN LOGS"
	if ui.ColorEnabled() {
		title = ui.Bold(ui.Cyan(title))
	}
	fmt.Fprintln(os.Stderr, title)
	fmt.Fprintln(os.Stderr, "Inspect the Warden audit log.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Usage:"))
	fmt.Fprintln(os.Stderr, "  warden logs [options]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Options:"))
	fmt.Fprintln(os.Stderr, "  --log <file>     Audit log path (default: ~/.local/state/warden/audit.jsonl)")
	fmt.Fprintln(os.Stderr, "  --tail <n>       Show last N lines")
	fmt.Fprintln(os.Stderr, "  --follow, -f     Follow/tail the log (polls every 250ms)")
	fmt.Fprintln(os.Stderr, "  --help           Show this help")
}

func printDoctorHelp() {
	title := "WARDEN DOCTOR"
	if ui.ColorEnabled() {
		title = ui.Bold(ui.Cyan(title))
	}
	fmt.Fprintln(os.Stderr, title)
	fmt.Fprintln(os.Stderr, "Check whether this system is ready to run Warden sandboxes.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Usage:"))
	fmt.Fprintln(os.Stderr, "  warden doctor")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Checks may include:"))
	fmt.Fprintln(os.Stderr, "  Sandbox backend")
	fmt.Fprintln(os.Stderr, "  Namespace support")
	fmt.Fprintln(os.Stderr, "  Network enforcement")
	fmt.Fprintln(os.Stderr, "  Required runtime dependencies")
	fmt.Fprintln(os.Stderr, "  Platform compatibility")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Security:"))
	fmt.Fprintln(os.Stderr, "  A failed security check must never be interpreted")
	fmt.Fprintln(os.Stderr, "  as 'sandbox disabled'. Warden fails closed.")
}

func printVersionHelp() {
	title := "WARDEN VERSION"
	if ui.ColorEnabled() {
		title = ui.Bold(ui.Cyan(title))
	}
	fmt.Fprintln(os.Stderr, title)
	fmt.Fprintln(os.Stderr, "Show the Warden version.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Usage:"))
	fmt.Fprintln(os.Stderr, "  warden version")
	fmt.Fprintln(os.Stderr, "  warden --version")
	fmt.Fprintln(os.Stderr, "  warden -v")
}

// printUpdateHelp prints help for `warden update`.
func printUpdateHelp() {
	title := "WARDEN UPDATE"
	if ui.ColorEnabled() {
		title = ui.Bold(ui.Cyan(title))
	}
	fmt.Fprintln(os.Stderr, title)
	fmt.Fprintln(os.Stderr, "Update warden using the npm registry + GitHub Releases.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Usage:"))
	fmt.Fprintln(os.Stderr, "  warden update")
	fmt.Fprintln(os.Stderr, "  warden update --check")
	fmt.Fprintln(os.Stderr, "  warden update --version <version>")
	fmt.Fprintln(os.Stderr, "  warden update --yes")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Flags:"))
	fmt.Fprintln(os.Stderr, "  --check              Report current vs latest without installing")
	fmt.Fprintln(os.Stderr, "  --version <version>  Install a specific published version")
	fmt.Fprintln(os.Stderr, "  --yes                Skip confirmation (required in non-TTY/CI)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("How it works:"))
	fmt.Fprintln(os.Stderr, "  Queries the npm registry for warden-sandbox-cli, downloads the")
	fmt.Fprintln(os.Stderr, "  matching GitHub Release binary for this platform, verifies its")
	fmt.Fprintln(os.Stderr, "  SHA256 against the published SHA256SUMS, and installs it into")
	fmt.Fprintln(os.Stderr, "  the versioned cache (~/.cache/warden/<ver>/). Fails closed on")
	fmt.Fprintln(os.Stderr, "  any verification problem.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Compatibility:"))
	fmt.Fprintln(os.Stderr, "  Releases published before `warden update` existed will not")
	fmt.Fprintln(os.Stderr, "  recognize this command. Upgrade those installs manually:")
	fmt.Fprintln(os.Stderr, "    npm install -g warden-sandbox-cli@latest")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Security:"))
	fmt.Fprintln(os.Stderr, "  HTTPS-only to pinned hosts. The downloaded binary is installed")
	fmt.Fprintln(os.Stderr, "  only after its checksum matches the release's SHA256SUMS file.")
	fmt.Fprintln(os.Stderr, "  No shell interpolation of version strings. Downgrades refused.")
}

// cmdUpdate runs the self-update flow and exits with a script-friendly code:
// 0 on success or when already current, 1 on failure.
func cmdUpdate(args []string) {
	opts, err := parseUpdateArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden update: %v\n", err)
		os.Exit(2)
	}
	// Ensure the stamped version, not the hardcoded dev default, is compared.
	selfupdate.SetCurrentVersion(version.Version)
	updated, err := selfupdate.RunOpts(os.Stdout, os.Stderr, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden update: %v\n", err)
		os.Exit(1)
	}
	_ = updated
}

func parseUpdateArgs(args []string) (selfupdate.Options, error) {
	var opts selfupdate.Options
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--check":
			opts.CheckOnly = true
		case arg == "--yes" || arg == "-y":
			opts.Yes = true
		case arg == "--version" || arg == "-version":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("flag %s requires a value", arg)
			}
			opts.TargetVersion = args[i+1]
			i++
		case strings.HasPrefix(arg, "--version="):
			opts.TargetVersion = strings.TrimPrefix(arg, "--version=")
		default:
			return opts, fmt.Errorf("unknown flag %q (want --check, --version, or --yes)", arg)
		}
	}
	if opts.CheckOnly && opts.TargetVersion != "" {
		return opts, fmt.Errorf("--check and --version cannot be combined")
	}
	return opts, nil
}

func maybePrintRunSummary(policyPath, backend string, p policy.Policy, cmd []string) {
	if os.Getenv("WARDEN_QUIET") != "" {
		return
	}
	if ui.IsCI() {
		return
	}
	if !ui.IsTerminalWriter(os.Stderr) {
		return
	}
	// Resolve backend display name
	be := backend
	if be == "" {
		be = "auto"
	}
	// Try to resolve actual backend for display, but don't fail if unavailable.
	if resolved, err := sandbox.Resolve(be); err == nil {
		be = resolved
	}
	title := "WARDEN"
	if ui.ColorEnabled() {
		title = ui.Bold(ui.Cyan(title))
	}
	fmt.Fprintln(os.Stderr, title)
	sep := "──────────────────────────────────────"
	if ui.TerminalWidth() < 50 {
		sep = "────────────────────────────────"
	}
	fmt.Fprintln(os.Stderr, ui.Dim(sep))
	fmt.Fprintln(os.Stderr, "")
	// Policy, backend, command
	if policyPath != "" {
		fmt.Fprintf(os.Stderr, "%-12s %s\n", ui.Dim("Policy"), ui.Cyan(policyPath))
	}
	fmt.Fprintf(os.Stderr, "%-12s %s\n", ui.Dim("Backend"), ui.Cyan(be))
	if len(cmd) > 0 {
		displayCmd := cmd[0]
		if len(cmd) > 1 {
			displayCmd += " " + strings.Join(cmd[1:], " ")
		}
		// Truncate long command for narrow terminals
		if len(displayCmd) > ui.TerminalWidth()-14 {
			displayCmd = displayCmd[:ui.TerminalWidth()-17] + "..."
		}
		fmt.Fprintf(os.Stderr, "%-12s %s\n", ui.Dim("Command"), displayCmd)
	}
	fmt.Fprintln(os.Stderr, "")
	// Filesystem grants
	fmt.Fprintln(os.Stderr, ui.Cyan("Filesystem"))
	if len(p.Filesystem.Read) == 0 && len(p.Filesystem.Write) == 0 {
		fmt.Fprintf(os.Stderr, "  %s %s\n", ui.Dim(ui.CrossMark()), ui.Dim("everything else (deny by default)"))
	} else {
		for _, r := range p.Filesystem.Read {
			fmt.Fprintf(os.Stderr, "  %s %s %s\n", ui.Green(ui.CheckMark()), r, ui.Dim("(read)"))
		}
		for _, w := range p.Filesystem.Write {
			fmt.Fprintf(os.Stderr, "  %s %s %s\n", ui.Green(ui.CheckMark()), w, ui.Dim("(write)"))
		}
		fmt.Fprintf(os.Stderr, "  %s %s\n", ui.Dim(ui.CrossMark()), ui.Dim("everything else"))
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Cyan("Network"))
	if len(p.Network.Allow) == 0 {
		fmt.Fprintf(os.Stderr, "  %s %s\n", ui.Dim(ui.CrossMark()), ui.Dim("everything else (deny by default)"))
	} else {
		for _, h := range p.Network.Allow {
			fmt.Fprintf(os.Stderr, "  %s %s\n", ui.Green(ui.CheckMark()), h)
		}
		fmt.Fprintf(os.Stderr, "  %s %s\n", ui.Dim(ui.CrossMark()), ui.Dim("everything else"))
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Cyan("Environment"))
	if len(p.Env.Allow) == 0 {
		fmt.Fprintf(os.Stderr, "  %s %s\n", ui.Dim(ui.CrossMark()), ui.Dim("all unspecified variables"))
	} else {
		for _, e := range p.Env.Allow {
			fmt.Fprintf(os.Stderr, "  %s %s\n", ui.Green(ui.CheckMark()), e)
		}
		fmt.Fprintf(os.Stderr, "  %s %s\n", ui.Dim(ui.CrossMark()), ui.Dim("all unspecified variables"))
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Dim(sep))
	// Sandbox active line — green, not just color
	fmt.Fprintf(os.Stderr, "%s %s\n", ui.Green(ui.CheckMark()), ui.Green("Sandbox active"))
	fmt.Fprintf(os.Stderr, "%s\n", ui.Dim("Warden fails closed when sandboxing is unavailable."))
	fmt.Fprintln(os.Stderr, "")
}

// cmdVersion prints the stamped build version and exits 0. Release builds
// stamp it via -ldflags (see the Makefile); builds from a source checkout
// without stamping report the in-source default.
// MUST remain script-friendly: single line "warden version X" with no banner.
func cmdVersion() {
	v := version.Version
	if v == "" {
		v = "dev"
	}
	fmt.Println("warden version", v)
}

func cmdRun(args []string) {
	approveEnabled, approveTimeout, rest, err := parseApproveFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		printRunHelp()
		os.Exit(2)
	}
	policyPath, backend, cmdTail, err := parseRunArgs(rest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		printRunHelp()
		os.Exit(2)
	}

	if approveEnabled {
		cmdRunWithApproval(policyPath, backend, cmdTail, approveTimeout)
		return // unreachable: cmdRunWithApproval always exits
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

	maybePrintRunSummary(policyPath, backend, p, cmd)

	exitCode, err := sandbox.Run(cmd, p, backend)
	if err != nil {
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

// maxApprovalRestarts caps the approve-mode respawn loop: every restart
// needs a fresh user approval, so hitting the cap means something is
// approving in a tight loop rather than a human deciding.
const maxApprovalRestarts = 10

// cmdRunWithApproval runs a server with interactive approval mode (M7) and
// respawns it when a filesystem approval requires a restart. The policy is
// reloaded from disk on every iteration so saved grants take effect. It
// always terminates the process via os.Exit.
func cmdRunWithApproval(policyPath, backend string, cmdTail []string, timeout time.Duration) {
	if err := approve.CheckTTY(); err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		os.Exit(2)
	}
	cfg := &approve.Config{Enabled: true, PolicyPath: policyPath, Timeout: timeout}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
		os.Exit(2)
	}
	for i := 0; i < maxApprovalRestarts; i++ {
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
		cmd, err = policy.ResolveExecutable(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
			os.Exit(2)
		}
		if err := checkSandboxCommand(cmd); err != nil {
			fmt.Fprintf(os.Stderr, "warden run: %v\n", err)
			os.Exit(2)
		}
		if i == 0 {
			maybePrintRunSummary(policyPath, backend, p, cmd)
		}
		exitCode, err := sandbox.RunWithApproval(cmd, p, backend, cfg)
		if errors.Is(err, approve.ErrRestartRequested) {
			fmt.Fprintf(os.Stderr, "warden run: restarting with the approved policy (%d/%d)\n", i+1, maxApprovalRestarts)
			continue
		}
		if err != nil {
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
	fmt.Fprintf(os.Stderr, "warden run: too many approval restarts (%d); inspect %s for runaway grants\n", maxApprovalRestarts, policyPath)
	os.Exit(1)
}

// checkSandboxCommand enforces the CLI-side contract every backend relies
// on: an absolute executable path that exists.
func checkSandboxCommand(cmd []string) error {
	// The sandbox needs an absolute path for the executable: backends bind
	// the parent dir, and a bare command name would resolve nowhere.
	if len(cmd) == 0 || !filepath.IsAbs(cmd[0]) {
		name := ""
		if len(cmd) > 0 {
			name = cmd[0]
		}
		return fmt.Errorf("command %q is not an absolute path; sandbox requires a full path to the executable", name)
	}
	if _, err := os.Stat(cmd[0]); err != nil {
		return fmt.Errorf("executable %q: %v", cmd[0], err)
	}
	return nil
}

// parseApproveFlags extracts --approve and --approve-timeout from run args,
// returning the remaining args for parseRunArgs. Unknown flags are left
// untouched for the downstream parser.
func parseApproveFlags(args []string) (enabled bool, timeout time.Duration, rest []string, err error) {
	rest = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--approve":
			enabled = true
		case strings.HasPrefix(arg, "--approve="):
			v := strings.TrimPrefix(arg, "--approve=")
			enabled, err = strconv.ParseBool(v)
			if err != nil {
				return false, 0, nil, fmt.Errorf("--approve=%q: want true or false", v)
			}
		case arg == "--approve-timeout":
			if i+1 >= len(args) {
				return false, 0, nil, fmt.Errorf("--approve-timeout requires a value (e.g. 2m)")
			}
			timeout, err = time.ParseDuration(args[i+1])
			if err != nil || timeout < 0 {
				return false, 0, nil, fmt.Errorf("--approve-timeout %q: want a non-negative duration like 30s or 2m", args[i+1])
			}
			i++
		case strings.HasPrefix(arg, "--approve-timeout="):
			v := strings.TrimPrefix(arg, "--approve-timeout=")
			timeout, err = time.ParseDuration(v)
			if err != nil || timeout < 0 {
				return false, 0, nil, fmt.Errorf("--approve-timeout %q: want a non-negative duration like 30s or 2m", v)
			}
		default:
			rest = append(rest, arg)
		}
	}
	if timeout > 0 && !enabled {
		return false, 0, nil, fmt.Errorf("--approve-timeout requires --approve")
	}
	return enabled, timeout, rest, nil
}

// parseRunArgs splits run flags into the policy path, optional backend, and
// command tail. A `--` separator is optional but supported: it lets the
// command start with a flag named like a warden option.
func parseRunArgs(args []string) (policyPath, backend string, cmd []string, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--policy" || arg == "-policy":
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("flag %s requires a value", arg)
			}
			if policyPath != "" {
				return "", "", nil, fmt.Errorf("--policy specified twice")
			}
			policyPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "--policy="):
			if policyPath != "" {
				return "", "", nil, fmt.Errorf("--policy specified twice")
			}
			policyPath = strings.TrimPrefix(arg, "--policy=")
		case arg == "--backend" || arg == "-backend":
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("flag %s requires a value", arg)
			}
			if backend != "" {
				return "", "", nil, fmt.Errorf("--backend specified twice")
			}
			backend = args[i+1]
			i++
		case strings.HasPrefix(arg, "--backend="):
			if backend != "" {
				return "", "", nil, fmt.Errorf("--backend specified twice")
			}
			backend = strings.TrimPrefix(arg, "--backend=")
		case arg == "--":
			cmd = args[i+1:]
			if policyPath == "" {
				return "", "", nil, fmt.Errorf("missing required flag --policy")
			}
			return policyPath, backend, cmd, nil
		case strings.HasPrefix(arg, "-"):
			return "", "", nil, fmt.Errorf("unknown flag %q (want --policy or --backend)", arg)
		default:
			// First non-flag argument: everything from here is the command.
			cmd = args[i:]
			if policyPath == "" {
				return "", "", nil, fmt.Errorf("missing required flag --policy")
			}
			return policyPath, backend, cmd, nil
		}
	}
	if policyPath == "" {
		return "", "", nil, fmt.Errorf("missing required flag --policy")
	}
	return policyPath, backend, nil, nil
}

func cmdTrace(args []string) {
	cmd, err := parseCommandArgs("trace", args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden trace: %v\n", err)
		os.Exit(2)
	}
	logFile, path, err := audit.OpenTrace()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden trace: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
	exitCode, err := audit.Trace(cmd, os.Environ(), os.Stdin, os.Stdout, os.Stderr, audit.New(logFile))
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden trace: %v\n", err)
		os.Exit(1)
	}
	// Keep run-like output clean; only report trace file to stderr.
	// Use semantic color for path.
	msg := fmt.Sprintf("warden trace: recorded access events in %s", path)
	if ui.ColorEnabled() {
		// Highlight path in cyan
		msg = fmt.Sprintf("warden trace: recorded access events in %s", ui.Cyan(path))
	}
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(exitCode)
}

func cmdInit(args []string) {
	logPath, outputPath, cmd, err := parseInitArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden init: %v\n", err)
		os.Exit(2)
	}
	// Polished presentation for init: show banner/header when TTY.
	isTTY := ui.IsTerminalWriter(os.Stderr) && !ui.IsCI()
	showBanner := isTTY
	if isTTY && ui.IsFirstRun() {
		// First-run welcome takes precedence; it already contains banner + intro.
		_, _ = ui.PrintWelcome(os.Stderr)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, ui.Dim("──────────────────────────────────────"))
		fmt.Fprintln(os.Stderr, "")
		_ = ui.MarkFirstRun()
		showBanner = false // avoid double banner
		what := "Let's secure your first MCP server."
		if ui.ColorEnabled() {
			what = ui.Bold(what)
		}
		fmt.Fprintln(os.Stderr, what)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, ui.Dim("What Warden does:"))
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "  %-15s %s\n", "Filesystem", ui.Dim("explicit paths only"))
		fmt.Fprintf(os.Stderr, "  %-15s %s\n", "Network", ui.Dim("explicit hosts only"))
		fmt.Fprintf(os.Stderr, "  %-15s %s\n", "Environment", ui.Dim("explicit variables only"))
		fmt.Fprintf(os.Stderr, "  %-15s %s\n", "Resources", ui.Dim("explicit limits"))
		fmt.Fprintln(os.Stderr, "")
	} else if showBanner {
		_, _ = ui.PrintBanner(os.Stderr)
		fmt.Fprintln(os.Stderr, "")
		what := "Let's secure your first MCP server."
		if ui.ColorEnabled() {
			what = ui.Bold(what)
		}
		fmt.Fprintln(os.Stderr, what)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, ui.Dim("What Warden does:"))
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "  %-15s %s\n", "Filesystem", ui.Dim("explicit paths only"))
		fmt.Fprintf(os.Stderr, "  %-15s %s\n", "Network", ui.Dim("explicit hosts only"))
		fmt.Fprintf(os.Stderr, "  %-15s %s\n", "Environment", ui.Dim("explicit variables only"))
		fmt.Fprintf(os.Stderr, "  %-15s %s\n", "Resources", ui.Dim("explicit limits"))
		fmt.Fprintln(os.Stderr, "")
	}
	if logPath == "" {
		logPath, err = audit.LatestTracePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warden init: %v\n", err)
			os.Exit(1)
		}
	}
	f, err := os.Open(logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden init: open audit log %q: %v\n", logPath, err)
		os.Exit(1)
	}
	events, readErr := audit.ReadEvents(f)
	closeErr := f.Close()
	if readErr != nil {
		fmt.Fprintf(os.Stderr, "warden init: %v\n", readErr)
		os.Exit(1)
	}
	if closeErr != nil {
		fmt.Fprintf(os.Stderr, "warden init: close audit log: %v\n", closeErr)
		os.Exit(1)
	}
	starter := policy.StarterFromAudit(events, cmd)
	data, err := starter.Marshal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden init: encode policy: %v\n", err)
		os.Exit(1)
	}
	if outputPath == "" {
		outputPath = "policy.yaml"
	}
	out, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			fmt.Fprintf(os.Stderr, "warden init: %q already exists; refusing to overwrite it\n", outputPath)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "warden init: create %q: %v\n", outputPath, err)
		os.Exit(1)
	}
	if _, err := out.Write(data); err != nil {
		out.Close()
		fmt.Fprintf(os.Stderr, "warden init: write %q: %v\n", outputPath, err)
		os.Exit(1)
	}
	if err := out.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "warden init: close %q: %v\n", outputPath, err)
		os.Exit(1)
	}
	// Polished success output.
	if isTTY {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, ui.Dim("Created:"))
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "  %s\n", ui.Cyan(outputPath))
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, ui.Dim("Next:"))
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "  %s\n", ui.Cyan(fmt.Sprintf("warden run --policy %s -- <server>", outputPath)))
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "%s Ready.\n", ui.Green(ui.CheckMark()))
		// Mark first-run as seen after successful init
		_ = ui.MarkFirstRun()
	} else {
		fmt.Fprintf(os.Stderr, "warden init: wrote conservative starter policy to %s\n", outputPath)
	}
}

func cmdLogs(args []string) {
	logPath, tail, follow, err := parseLogsArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden logs: %v\n", err)
		os.Exit(2)
	}
	if logPath == "" {
		logPath, err = audit.DefaultPath()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden logs: %v\n", err)
		os.Exit(1)
	}
	f, err := os.Open(logPath)
	if os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "warden logs: no audit events recorded yet")
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden logs: %v\n", err)
		os.Exit(1)
	}
	data, err := io.ReadAll(f)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden logs: %v\n", err)
		os.Exit(1)
	}
	followOffset := int64(len(data))
	if tail > 0 {
		data = lastLines(data, tail)
	}
	if _, err := os.Stdout.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "warden logs: %v\n", err)
		os.Exit(1)
	}
	if follow {
		if err := followLog(logPath, followOffset); err != nil {
			fmt.Fprintf(os.Stderr, "warden logs: %v\n", err)
			os.Exit(1)
		}
	}
}

func parseCommandArgs(name string, args []string) ([]string, error) {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: warden %s -- <command...>", name)
	}
	return args, nil
}

func parseInitArgs(args []string) (logPath, outputPath string, cmd []string, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--log":
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("--log requires a value")
			}
			logPath, i = args[i+1], i+1
		case "--output", "-o":
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("%s requires a value", args[i])
			}
			outputPath, i = args[i+1], i+1
		case "--":
			return logPath, outputPath, args[i+1:], nil
		default:
			return "", "", nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return logPath, outputPath, nil, nil
}

func parseLogsArgs(args []string) (logPath string, tail int, follow bool, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--log":
			if i+1 >= len(args) {
				return "", 0, false, fmt.Errorf("--log requires a value")
			}
			logPath, i = args[i+1], i+1
		case "--tail", "-n":
			if i+1 >= len(args) {
				return "", 0, false, fmt.Errorf("%s requires a value", args[i])
			}
			tail, err = strconv.Atoi(args[i+1])
			if err != nil || tail < 0 {
				return "", 0, false, fmt.Errorf("%s must be a non-negative integer", args[i])
			}
			i++
		case "--follow", "-f":
			follow = true
		default:
			return "", 0, false, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return logPath, tail, follow, nil
}

func lastLines(data []byte, n int) []byte {
	data = bytes.TrimRight(data, "\n")
	if len(data) == 0 || n == 0 {
		return data
	}
	for i := len(data) - 1; i >= 0; i-- {
		if data[i] == '\n' {
			n--
			if n == 0 {
				return append(data[i+1:], '\n')
			}
		}
	}
	return append(data, '\n')
}

func followLog(path string, offset int64) error {
	// Polling avoids platform-specific file-watch dependencies and handles a
	// writer appending JSON Lines while `warden logs -f` is running.
	for {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Size() < offset {
			offset = 0
		} // log rotation/truncation
		if info.Size() > offset {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			if _, err := f.Seek(offset, io.SeekStart); err != nil {
				f.Close()
				return err
			}
			n, err := io.Copy(os.Stdout, f)
			closeErr := f.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
			offset += n
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func cmdProxyBridge(args []string) {
	socket, listen, target, err := proxy.ParseBridgeArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden proxy bridge: %v\n", err)
		os.Exit(2)
	}
	code, err := proxy.RunBridge(socket, listen, target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden proxy bridge: %v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func printProxyHelp() {
	title := "WARDEN PROXY"
	if ui.ColorEnabled() {
		title = ui.Bold(ui.Cyan(title))
	}
	fmt.Fprintln(os.Stderr, title)
	fmt.Fprintln(os.Stderr, "Run MCP client proxy with policy-based filtering and auditing.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Usage:"))
	fmt.Fprintln(os.Stderr, "  warden proxy --policy <file> [options]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Options:"))
	fmt.Fprintln(os.Stderr, "  --policy <file>            MCP proxy policy file (required)")
	fmt.Fprintln(os.Stderr, "  --listen <addr>            Listen address (default: localhost:8765)")
	fmt.Fprintln(os.Stderr, "  --upstream <url>           Override upstream from policy")
	fmt.Fprintln(os.Stderr, "  --help                     Show this help")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Examples:"))
	fmt.Fprintln(os.Stderr, "  warden proxy --policy mcp-policy.yaml")
	fmt.Fprintln(os.Stderr, "  warden proxy --policy mcp-policy.yaml --listen :9000")
	fmt.Fprintln(os.Stderr, "  warden proxy --policy mcp-policy.yaml --upstream \"stdio:cat\"")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Policy Format:"))
	fmt.Fprintln(os.Stderr, "  mcp:")
	fmt.Fprintln(os.Stderr, "    upstream: \"stdio:npx @modelcontextprotocol/server-github\"")
	fmt.Fprintln(os.Stderr, "    allow_tools: [\"list_repos\", \"get_file\"]")
	fmt.Fprintln(os.Stderr, "    deny_patterns: [\"ghp_[A-Za-z0-9]{36}\"]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Security:"))
	fmt.Fprintln(os.Stderr, "  The proxy filters newline-delimited JSON-RPC messages in both directions:")
	fmt.Fprintln(os.Stderr, "  tool allowlist, deny patterns, and payload size are enforced and audited.")
	fmt.Fprintln(os.Stderr, "  • Upstreams: stdio commands or http(s):// MCP servers (Streamable HTTP")
	fmt.Fprintln(os.Stderr, "    and SSE). Plain http:// is allowed only for loopback dev servers;")
	fmt.Fprintln(os.Stderr, "    remote upstreams must use https://.")
	fmt.Fprintln(os.Stderr, "  • A stdio subprocess receives only env.allow variables (deny-by-default),")
	fmt.Fprintln(os.Stderr, "    but is NOT itself sandboxed. Wrap `warden proxy` in `warden run` if the")
	fmt.Fprintln(os.Stderr, "    upstream needs filesystem/network isolation.")
}

func cmdProxy(args []string) {
	policyPath, listen, upstreamOverride, err := parseProxyArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden proxy: %v\n", err)
		printProxyHelp()
		os.Exit(2)
	}

	p, err := policy.Load(policyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden proxy: %v\n", err)
		os.Exit(2)
	}

	if p.MCP == nil {
		fmt.Fprintf(os.Stderr, "warden proxy: policy file must contain 'mcp' section\n")
		os.Exit(2)
	}

	// Override upstream if provided via command line
	if upstreamOverride != "" {
		p.MCP.Upstream = upstreamOverride
	}

	if p.MCP.Upstream == "" {
		fmt.Fprintf(os.Stderr, "warden proxy: policy must specify mcp.upstream\n")
		os.Exit(2)
	}

	// Set up audit logging
	auditFile, auditPath, err := audit.OpenDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden proxy: failed to set up audit logging: %v\n", err)
		os.Exit(1)
	}
	defer auditFile.Close()

	// Build the MCP proxy policy from the loaded policy, honoring the CLI
	// override for --upstream. Only local stdio upstreams are supported;
	// HTTP/SSE upstreams are rejected by NewProxyServer (not implemented).
	mcpPolicy := mcpproxy.MCPPolicy{
		Upstream:      p.MCP.Upstream,
		AllowTools:    p.MCP.AllowTools,
		DenyPatterns:  p.MCP.DenyPatterns,
		MaxPayloadKB:  p.MCP.MaxPayloadKB,
		AuditRequests: p.MCP.AuditRequests,
		EnvAllow:      p.EnvAllowlist(),
	}

	server, err := mcpproxy.NewProxyServer(mcpPolicy, audit.New(auditFile))
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden proxy: %v\n", err)
		os.Exit(1)
	}
	defer server.Close()

	if err := server.Start(listen); err != nil {
		fmt.Fprintf(os.Stderr, "warden proxy: failed to start: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "✓ warden proxy listening on %s\n", server.Addr())
	fmt.Fprintf(os.Stderr, "  upstream:  %s\n", p.MCP.Upstream)
	if len(p.MCP.AllowTools) > 0 {
		fmt.Fprintf(os.Stderr, "  tools:     %v\n", p.MCP.AllowTools)
	}
	if len(p.MCP.DenyPatterns) > 0 {
		fmt.Fprintf(os.Stderr, "  deny:      %v\n", p.MCP.DenyPatterns)
	}
	fmt.Fprintf(os.Stderr, "  audit log: %s\n", auditPath)
	fmt.Fprintf(os.Stderr, "  (ctrl-C to stop)\n")
	if strings.HasPrefix(p.MCP.Upstream, "stdio:") {
		fmt.Fprintf(os.Stderr, "  ⚠ experimental: the stdio subprocess gets only env.allow variables (deny-by-default),\n")
		fmt.Fprintf(os.Stderr, "    but its filesystem and network access are NOT sandboxed; Warden filters and\n")
		fmt.Fprintf(os.Stderr, "    audits the JSON-RPC messages in both directions.\n")
	} else {
		fmt.Fprintf(os.Stderr, "  ⚠ experimental: HTTP/SSE filtering is message-level (tool allowlist, deny\n")
		fmt.Fprintf(os.Stderr, "    patterns, payload size). TLS is verified; plain http:// is loopback-only.\n")
	}

	// Block until interrupted. The proxy server goroutines shut down with
	// the process when this returns to main.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Fprintln(os.Stderr, "\nwarden proxy: shutting down")
}

func parseProxyArgs(args []string) (policyPath, listen, upstream string, err error) {
	listen = "localhost:8765" // default

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--policy" || arg == "-policy":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("flag %s requires a value", arg)
			}
			if policyPath != "" {
				return "", "", "", fmt.Errorf("--policy specified twice")
			}
			policyPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "--policy="):
			if policyPath != "" {
				return "", "", "", fmt.Errorf("--policy specified twice")
			}
			policyPath = strings.TrimPrefix(arg, "--policy=")
		case arg == "--listen" || arg == "-listen":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("flag %s requires a value", arg)
			}
			listen = args[i+1]
			i++
		case strings.HasPrefix(arg, "--listen="):
			listen = strings.TrimPrefix(arg, "--listen=")
		case arg == "--upstream" || arg == "-upstream":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("flag %s requires a value", arg)
			}
			upstream = args[i+1]
			i++
		case strings.HasPrefix(arg, "--upstream="):
			upstream = strings.TrimPrefix(arg, "--upstream=")
		case strings.HasPrefix(arg, "-"):
			return "", "", "", fmt.Errorf("unknown flag %q (want --policy, --listen, or --upstream)", arg)
		default:
			return "", "", "", fmt.Errorf("unexpected argument %q", arg)
		}
	}

	if policyPath == "" {
		return "", "", "", fmt.Errorf("missing required flag --policy")
	}

	return policyPath, listen, upstream, nil
}

func printK8sHelp() {
	title := "WARDEN K8S"
	if ui.ColorEnabled() {
		title = ui.Bold(ui.Cyan(title))
	}
	fmt.Fprintln(os.Stderr, title)
	fmt.Fprintln(os.Stderr, "Generate container and Kubernetes manifests from Warden policies.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Usage:"))
	fmt.Fprintln(os.Stderr, "  warden k8s <command> --policy <file> [options]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Commands:"))
	fmt.Fprintln(os.Stderr, "  render       Generate K8s YAML manifests")
	fmt.Fprintln(os.Stderr, "  docker       Generate Docker run command")
	fmt.Fprintln(os.Stderr, "  validate     Validate policy for container deployment")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Options:"))
	fmt.Fprintln(os.Stderr, "  --policy <file>        Warden policy file (required)")
	fmt.Fprintln(os.Stderr, "  --image <image>        Container image to use")
	fmt.Fprintln(os.Stderr, "  --namespace <ns>       K8s namespace (default: default)")
	fmt.Fprintln(os.Stderr, "  --output <file>        Output file (default: stdout)")
	fmt.Fprintln(os.Stderr, "  --platform <platform>  Target platform (docker, kubernetes)")
	fmt.Fprintln(os.Stderr, "  --help                 Show this help")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Examples:"))
	fmt.Fprintln(os.Stderr, "  warden k8s render --policy policy.yaml --image myapp:latest")
	fmt.Fprintln(os.Stderr, "  warden k8s docker --policy policy.yaml --image myapp:latest")
	fmt.Fprintln(os.Stderr, "  warden k8s validate --policy policy.yaml")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Policy Translation:"))
	fmt.Fprintln(os.Stderr, "  filesystem.read    → readOnlyRootFilesystem + volume mounts")
	fmt.Fprintln(os.Stderr, "  filesystem.write   → emptyDir/hostPath volumes")
	fmt.Fprintln(os.Stderr, "  network.allow      → NetworkPolicy egress rules")
	fmt.Fprintln(os.Stderr, "  env.allow          → container env variables")
	fmt.Fprintln(os.Stderr, "  limits             → resource limits/requests")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, ui.Bold("Security:"))
	fmt.Fprintln(os.Stderr, "  Generated manifests include:")
	fmt.Fprintln(os.Stderr, "  - readOnlyRootFilesystem: true")
	fmt.Fprintln(os.Stderr, "  - runAsNonRoot: true")
	fmt.Fprintln(os.Stderr, "  - capabilities: drop ALL")
	fmt.Fprintln(os.Stderr, "  - seccompProfile: RuntimeDefault")
}

func cmdK8s(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "warden k8s: missing command (render, docker, validate)\n")
		printK8sHelp()
		os.Exit(2)
	}

	command := args[0]
	policyPath, image, namespace, outputFile, platform, err := parseK8sArgs(args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden k8s: %v\n", err)
		printK8sHelp()
		os.Exit(2)
	}

	p, err := policy.Load(policyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden k8s: %v\n", err)
		os.Exit(2)
	}

	switch command {
	case "render":
		cmdK8sRender(p, image, namespace, outputFile, platform)
	case "docker":
		cmdK8sDocker(p, image, platform)
	case "validate":
		cmdK8sValidate(p)
	default:
		fmt.Fprintf(os.Stderr, "warden k8s: unknown command %q (want render, docker, validate)\n", command)
		os.Exit(2)
	}
}

func cmdK8sRender(p policy.Policy, image, namespace, outputFile, platform string) {
	if image == "" {
		fmt.Fprintf(os.Stderr, "warden k8s: render requires --image\n")
		os.Exit(2)
	}

	opts := container.TranslateOptions{
		Image:        image,
		Namespace:    namespace,
		Platform:     "kubernetes",
		EmitWarnings: true,
	}

	manifests, err := container.GenerateKubernetesManifests(p, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden k8s: %v\n", err)
		os.Exit(1)
	}

	out, err := container.RenderKubernetesYAML(manifests)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden k8s: %v\n", err)
		os.Exit(1)
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(out), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "warden k8s: write output: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ wrote Kubernetes manifests to %s\n", outputFile)
	} else {
		fmt.Print(out)
	}

	// Surface FQDN-limitation warnings on stderr so they are visible without
	// contaminating the YAML written to stdout/file.
	for _, w := range container.ValidateKubernetesPolicy(p) {
		fmt.Fprintf(os.Stderr, "⚠️  %s\n", w)
	}
}

func cmdK8sDocker(p policy.Policy, image, platform string) {
	if image == "" {
		fmt.Fprintf(os.Stderr, "warden k8s: docker requires --image\n")
		os.Exit(2)
	}

	cmd, err := container.RenderDockerCommand(p, container.TranslateOptions{Image: image, Platform: platform})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden k8s: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(strings.Join(cmd, " ") + "\n")
}

func cmdK8sValidate(p policy.Policy) {
	fmt.Printf("🔍 Validating policy for container deployment...\n\n")

	// Check basic structure
	fmt.Printf("📋 Policy Structure:\n")
	if len(p.Command) > 0 {
		fmt.Printf("✅ Command specified: %v\n", p.Command)
	} else {
		fmt.Printf("⚠️  No command specified - will need to be provided at runtime\n")
	}

	fmt.Printf("✅ Filesystem: %d read, %d write paths\n", len(p.Filesystem.Read), len(p.Filesystem.Write))
	fmt.Printf("✅ Network: %d allowed hosts\n", len(p.Network.Allow))
	fmt.Printf("✅ Environment: %d allowed variables\n", len(p.Env.Allow))

	// Delegate container-compatibility checks to the container package so the
	// CLI and `go test` exercise the same translation logic.
	fmt.Printf("\n🔧 Container Compatibility:\n")
	warnings := container.ValidateKubernetesPolicy(p)
	for _, w := range warnings {
		fmt.Printf("⚠️  %s\n", w)
	}
	if p.Limits.MemoryMB == 0 {
		fmt.Printf("⚠️  No memory limit specified - recommended for K8s deployment\n")
	}
	if len(warnings) == 0 && p.Limits.MemoryMB > 0 {
		fmt.Printf("✅ Policy is fully compatible with container deployment\n")
	}

	fmt.Printf("\n🛡️  Security Assessment:\n")
	fmt.Printf("✅ Read-only root filesystem will be enforced\n")
	fmt.Printf("✅ Capability dropping will be applied\n")
	fmt.Printf("✅ Non-root execution will be enforced\n")
	fmt.Printf("✅ Seccomp filtering will be applied\n")

	fmt.Printf("\n🎯 Deployment Readiness:\n")
	if len(warnings) > 0 || p.Limits.MemoryMB == 0 {
		fmt.Printf("⚠️  Policy has compatibility warnings - review before deployment\n")
	} else {
		fmt.Printf("✅ Policy is ready for container deployment\n")
	}
}

func parseK8sArgs(args []string) (policyPath, image, namespace, outputFile, platform string, err error) {
	namespace = "default"   // default
	platform = "kubernetes" // default

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--policy" || arg == "-policy":
			if i+1 >= len(args) {
				return "", "", "", "", "", fmt.Errorf("flag %s requires a value", arg)
			}
			if policyPath != "" {
				return "", "", "", "", "", fmt.Errorf("--policy specified twice")
			}
			policyPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "--policy="):
			if policyPath != "" {
				return "", "", "", "", "", fmt.Errorf("--policy specified twice")
			}
			policyPath = strings.TrimPrefix(arg, "--policy=")
		case arg == "--image":
			if i+1 >= len(args) {
				return "", "", "", "", "", fmt.Errorf("flag %s requires a value", arg)
			}
			image = args[i+1]
			i++
		case strings.HasPrefix(arg, "--image="):
			image = strings.TrimPrefix(arg, "--image=")
		case arg == "--namespace":
			if i+1 >= len(args) {
				return "", "", "", "", "", fmt.Errorf("flag %s requires a value", arg)
			}
			namespace = args[i+1]
			i++
		case strings.HasPrefix(arg, "--namespace="):
			namespace = strings.TrimPrefix(arg, "--namespace=")
		case arg == "--output":
			if i+1 >= len(args) {
				return "", "", "", "", "", fmt.Errorf("flag %s requires a value", arg)
			}
			outputFile = args[i+1]
			i++
		case strings.HasPrefix(arg, "--output="):
			outputFile = strings.TrimPrefix(arg, "--output=")
		case arg == "--platform":
			if i+1 >= len(args) {
				return "", "", "", "", "", fmt.Errorf("flag %s requires a value", arg)
			}
			platform = args[i+1]
			i++
		case strings.HasPrefix(arg, "--platform="):
			platform = strings.TrimPrefix(arg, "--platform=")
		case strings.HasPrefix(arg, "-"):
			return "", "", "", "", "", fmt.Errorf("unknown flag %q", arg)
		default:
			return "", "", "", "", "", fmt.Errorf("unexpected argument %q", arg)
		}
	}

	if policyPath == "" {
		return "", "", "", "", "", fmt.Errorf("missing required flag --policy")
	}

	return policyPath, image, namespace, outputFile, platform, nil
}
