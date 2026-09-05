// Command warden is the CLI entry point for the Warden sandbox runtime.
//
// Subcommands:
//   - `warden run --policy <file> -- <command...>` runs a server under a
//     policy, sandboxed by the current OS backend (Linux: bubblewrap).
//   - `warden trace`, `warden init`, and `warden logs` are usability tools.
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/proxy"
	"github.com/warden-sandbox/warden/internal/sandbox"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "__proxy-bridge":
		cmdProxyBridge(os.Args[2:])
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
  warden trace -- <command...>                 Run unsandboxed and record access attempts
  warden init [--log <file>] [--output <file>] [-- <command...>]
                                                Generate a starter policy from an audit log
  warden logs [--tail <n>] [--follow] [--log <file>]
                                                Inspect or follow the audit log

See ROADMAP.md. M1–M3 are implemented on Linux.`)
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

	exitCode, err := sandbox.Run(cmd, p)
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
	fmt.Fprintf(os.Stderr, "warden trace: recorded access events in %s\n", path)
	os.Exit(exitCode)
}

func cmdInit(args []string) {
	logPath, outputPath, cmd, err := parseInitArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden init: %v\n", err)
		os.Exit(2)
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
	fmt.Fprintf(os.Stderr, "warden init: wrote conservative starter policy to %s\n", outputPath)
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
	var socket, listen string
	i := 0
	for i < len(args) && args[i] != "--" {
		if i+1 >= len(args) {
			fmt.Fprintln(os.Stderr, "warden proxy bridge: flag requires a value")
			os.Exit(2)
		}
		switch args[i] {
		case "--socket":
			socket = args[i+1]
		case "--listen":
			listen = args[i+1]
		default:
			fmt.Fprintf(os.Stderr, "warden proxy bridge: unknown flag %q\n", args[i])
			os.Exit(2)
		}
		i += 2
	}
	if i == len(args) || socket == "" || listen == "" || i+1 == len(args) {
		fmt.Fprintln(os.Stderr, "warden proxy bridge: missing socket, listen address, or command")
		os.Exit(2)
	}
	code, err := proxy.RunBridge(socket, listen, args[i+1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "warden proxy bridge: %v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}
