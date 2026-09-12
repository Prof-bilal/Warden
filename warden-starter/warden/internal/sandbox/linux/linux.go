//go:build linux

// Package linux implements the Linux sandbox backend using bubblewrap
// (bwrap). The sandbox is deny-by-default: the process sees only the
// runtime base (/usr, /lib64), pseudo-filesystems, and the paths granted by
// the policy.
package linux

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/envfilter"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/proxy"
	"github.com/warden-sandbox/warden/internal/sandbox/sandboxerr"
)

// runtimeBase are read-only mounts every sandboxed process gets, because a
// dynamically linked binary cannot even start without them. On merged-/usr
// distributions /lib64 is a symlink into /usr/lib64 and bwrap does not
// follow symlinks across bind mounts, so each base path must be mounted
// explicitly at its own location.
var runtimeBase = []string{"/usr", "/lib64"}

// BuildBwrapArgs translates a policy plus a resolved command into the
// complete bwrap argument list. It is a pure function so the mount logic
// can be unit-tested without bwrap installed.
//
// The returned slice includes everything up to the command: the caller must
// append the command and its arguments after it.
func BuildBwrapArgs(cmd []string, p policy.Policy) ([]string, error) {
	if len(cmd) == 0 {
		return nil, fmt.Errorf("bwrap args: no command")
	}

	exe := cmd[0]
	if !filepath.IsAbs(exe) {
		return nil, fmt.Errorf("bwrap args: command %q must be an absolute path", exe)
	}

	seen := make(map[string]bool)
	var args []string
	add := func(flag, src string) {
		// Under /tmp the fresh tmpfs is already mounted: give the bind an
		// explicit mountpoint first, or the tmpfs swallows it.
		if strings.HasPrefix(src, "/tmp/") {
			args = append(args, "--dir", src)
		}
		args = append(args, flag, src, src)
		seen[src] = true
	}

	// Namespace isolation: the process gets its own user/ipc/pid/net
	// namespaces. Network egress policy is M2; --unshare-net already cuts
	// the process off from the host network.
	args = append(args,
		"--unshare-user", "--unshare-ipc", "--unshare-pid", "--unshare-net",
		"--disable-userns", "--die-with-parent",
		"--uid", "0", "--gid", "0",
	)

	// Dev and proc first: later binds under them are rare, and bwrap applies
	// mounts in order. /tmp is replaced with a fresh writable tmpfs BEFORE
	// any path under it is mounted, so a bind to /tmp/foo needs a --dir
	// mountpoint inside the tmpfs (bwrap applies mounts in order and a
	// tmpfs mounted later would hide the earlier bind).
	args = append(args, "--dev", "/dev", "--proc", "/proc", "--size", "67108864", "--tmpfs", "/tmp")

	// Runtime base (read-only).
	for _, base := range runtimeBase {
		add("--ro-bind", base)
	}

	// Policy grants. A path granted both read and write is ambiguous —
	// refuse it rather than let mount order decide.
	mode := make(map[string]string)
	for _, path := range p.Filesystem.Read {
		if mode[path] == "write" {
			return nil, fmt.Errorf("bwrap args: %q is granted as both read and write", path)
		}
		mode[path] = "read"
		if !seen[path] {
			add("--ro-bind", path)
		}
	}
	for _, path := range p.Filesystem.Write {
		if mode[path] == "read" {
			return nil, fmt.Errorf("bwrap args: %q is granted as both read and write", path)
		}
		// The runtime base is always read-only; a write grant into it would
		// remount part of the system as writable and widen the sandbox.
		for _, base := range runtimeBase {
			if path == base || isUnder(path, base) {
				return nil, fmt.Errorf("bwrap args: %q is inside the read-only runtime base %s and cannot be granted write", path, base)
			}
		}
		mode[path] = "write"
		add("--bind", path)
	}

	// If the command's directory is not already mounted (e.g. a binary
	// outside /usr, like a mise-installed node), bind its parent dir so the
	// executable stays visible.
	parent := filepath.Dir(exe)
	if !seen[parent] {
		add("--ro-bind", parent)
	}

	return args, nil
}

// isUnder reports whether path is inside dir (strictly under it). Both must
// be absolute; lexical clean is applied so "./a/.." cases are handled.
func isUnder(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Run spawns the command inside a bubblewrap sandbox built from the policy.
// Stdio is passed through directly: the sandboxed process's stdin/stdout/
// stderr are the parent's, so an MCP client talking over stdio sees no
// protocol-level difference.
//
// It returns the sandboxed process's exit code, or an error if the process
// could not be started at all. A missing bwrap is an error — Warden never
// falls back to running the command unsandboxed.
func Run(cmd []string, p policy.Policy) (int, error) {
	// Check the core backend before opening persistent state so a missing
	// bwrap always reports the real, actionable failure.
	if _, err := exec.LookPath("bwrap"); err != nil {
		return 0, sandboxerr.RefuseToRun{Reason: "bubblewrap (bwrap) is not installed on this Linux host"}
	}
	logFile, _, err := audit.OpenDefault()
	if err != nil {
		return 0, fmt.Errorf("open audit log: %w", err)
	}
	defer logFile.Close()
	return runWithEnvAndAudit(cmd, p, os.Environ(), audit.New(logFile), nil)
}

func runWithEnv(cmd []string, p policy.Policy, parentEnv []string) (int, error) {
	return runWithEnvAndAudit(cmd, p, parentEnv, nil, nil)
}

// RunWithApproval spawns the command like Run but with interactive approval
// mode (M7): blocked network requests prompt on the terminal and apply live
// via the egress proxy, while blocked file accesses detected in the live
// strace stream prompt with an offer to save the grant and restart. A nil or
// disabled cfg behaves exactly like Run.
func RunWithApproval(cmd []string, p policy.Policy, cfg *approve.Config) (int, error) {
	if cfg == nil || !cfg.Enabled {
		return Run(cmd, p)
	}
	if err := cfg.Validate(); err != nil {
		return 0, err
	}
	logFile, _, err := audit.OpenDefault()
	if err != nil {
		return 0, fmt.Errorf("open audit log: %w", err)
	}
	defer logFile.Close()
	return runWithEnvAndAudit(cmd, p, os.Environ(), audit.New(logFile), cfg)
}

func runWithEnvAndAudit(cmd []string, p policy.Policy, parentEnv []string, logger *audit.Logger, approval *approve.Config) (int, error) {
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		return 0, sandboxerr.RefuseToRun{Reason: "bubblewrap (bwrap) is not installed on this Linux host"}
	}
	setpriv, err := exec.LookPath("setpriv")
	if err != nil {
		return 0, sandboxerr.RefuseToRun{Reason: "setpriv is not installed; it is required to enforce no_new_privs"}
	}

	args, err := BuildBwrapArgs(cmd, p)
	if err != nil {
		return 0, fmt.Errorf("build bwrap args: %w", err)
	}
	// The host-side proxy has the only external network socket.  The target
	// gets a fresh network namespace (above), where direct connections have
	// no route; it can only reach the loopback bridge below.
	eg, err := proxy.Start(p.Network.Allow, logger)
	if err != nil {
		return 0, fmt.Errorf("start egress proxy: %w", err)
	}
	defer eg.Close()

	// Interactive approval mode: blocked hosts prompt on the terminal and
	// approved ones join the proxy allowlist while the server keeps running.
	var prompter *approve.Prompter
	if approval != nil && approval.Enabled {
		prompter = approve.NewPrompter(approval.PolicyPath, approval.Timeout, logger)
		eg.SetApprover(prompter.NetworkApprover())
	}
	bridgeExe, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("locate proxy bridge executable: %w", err)
	}
	const bridgePath = "/.warden/proxy-bridge"
	const proxyDir = "/.warden/host-proxy"
	const socketPath = proxyDir + "/egress.sock"
	args = append(args,
		"--ro-bind", bridgeExe, bridgePath,
		// Bind the socket's parent directory rather than a socket inode:
		// Linux reliably bind-mounts directories, while socket-file bind
		// mounts vary by kernel. The directory has mode 0700 and contains
		// only this one read-only proxy endpoint.
		"--ro-bind", filepath.Dir(eg.SocketPath()), proxyDir,
	)

	bridgeArgs := []string{bridgePath, "__proxy-bridge", "--socket", socketPath, "--listen", "127.0.0.1:18080", "--"}
	bridgeArgs = append(bridgeArgs, cmd...)
	runArgs := append(args, bridgeArgs...)
	program := setpriv
	runArgs = append([]string{"--nnp", bwrap}, runArgs...)
	tracePath := ""
	if logger != nil {
		strace, err := exec.LookPath("strace")
		if err != nil {
			return 0, fmt.Errorf("strace not found: required for complete file/network auditing on Linux: %w", err)
		}
		trace, path, err := audit.OpenRawTrace()
		if err != nil {
			return 0, fmt.Errorf("create audit trace: %w", err)
		}
		tracePath = path
		if err := trace.Close(); err != nil {
			return 0, fmt.Errorf("close audit trace: %w", err)
		}
		defer os.Remove(tracePath)
		program = strace
		runArgs = append([]string{"-f", "-qq", "-s", "4096", "-e", "trace=%file,%network", "-o", tracePath, setpriv}, runArgs...)
	}
	sub := exec.Command(program, runArgs...)
	sub.Stdin = os.Stdin
	sub.Stdout = os.Stdout
	sub.Stderr = os.Stderr
	// A dedicated group lets M3 cleanly terminate all descendants when a
	// policy limit is breached without ever signalling Warden itself.
	sub.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// Deny by default for the environment: only policy-allowlisted names
	// are forwarded from the parent.
	sub.Env = append(envfilter.Filter(parentEnv, p.EnvAllowlist()),
		"HTTP_PROXY=http://127.0.0.1:18080",
		"HTTPS_PROXY=http://127.0.0.1:18080",
		"ALL_PROXY=http://127.0.0.1:18080",
		"NO_PROXY=",
	)

	if err := sub.Start(); err != nil {
		return 0, fmt.Errorf("start sandboxed process: %w", err)
	}

	// Filesystem approval watches the live strace stream for denials worth
	// prompting about. Bind mounts are fixed at spawn, so an approved grant
	// is saved to the policy file and the watcher cancels the run context,
	// which the wait below turns into a restart signal for the CLI.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if prompter != nil {
		if tracePath == "" {
			_ = logger.Log(audit.Event{Type: "approval", Action: "unavailable", Resource: "filesystem", Allowed: false, Reason: "no audit trace; filesystem approvals disabled for this run"})
		} else {
			go watchFileApprovals(ctx, prompter, approval.PolicyPath, tracePath, p, cmd, logger, cancel)
		}
	}
	var runErr error
	var limitErr *LimitExceededError
	if prompter != nil {
		var restarted bool
		runErr, limitErr, restarted = waitWithLimitsCtx(sub, p.Limits, ctx)
		if restarted {
			// Preserve the partial trace as evidence, then ask the CLI
			// to respawn under the widened policy file.
			if tracePath != "" {
				_ = importTraceFile(tracePath, logger)
			}
			return 0, approve.ErrRestartRequested
		}
	} else {
		runErr, limitErr = waitWithLimits(sub, p.Limits)
	}
	if tracePath != "" {
		if err := importTraceFile(tracePath, logger); err != nil {
			return 0, err
		}
	}
	if limitErr != nil {
		_ = logger.Log(audit.Event{Type: "limit", Action: "terminate", Resource: limitErr.Limit, Allowed: false, Reason: limitErr.Error()})
		return 0, limitErr
	}
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 0, fmt.Errorf("run sandboxed process: %w", runErr)
	}
	return 0, nil
}

// importTraceFile folds a completed strace log into the audit logger.
func importTraceFile(tracePath string, logger *audit.Logger) error {
	f, err := os.Open(tracePath)
	if err != nil {
		return fmt.Errorf("open completed audit trace: %w", err)
	}
	importErr := audit.ImportStrace(f, logger)
	closeErr := f.Close()
	if importErr != nil {
		return fmt.Errorf("import complete audit trace: %w", importErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close completed audit trace: %w", closeErr)
	}
	return nil
}

// watchFileApprovals tails the live strace log and prompts for denials that
// qualify via approve.ShouldPromptFile. Approved grants are saved to the
// policy file immediately; a restart choice invokes requestRestart (the run
// context cancel), which ends the wait so the CLI can respawn. The policy
// snapshot refreshes after every save so later prompts see earlier grants.
func watchFileApprovals(ctx context.Context, prompter *approve.Prompter, policyPath, tracePath string, pol policy.Policy, cmd []string, logger *audit.Logger, requestRestart func()) {
	var mu sync.Mutex // guards snapshot; emit is single-threaded but be explicit
	snapshot := pol
	exe := ""
	if len(cmd) > 0 {
		exe = cmd[0]
	}
	emit := func(ev audit.Event) {
		mu.Lock()
		snap := snapshot
		mu.Unlock()
		grant, write, ok := approve.ShouldPromptFile(ev, &snap, exe)
		if !ok {
			return
		}
		outcome, fresh := prompter.PromptFile(ev.Action, ev.Resource, grant, write)
		if !fresh || outcome == approve.FileDeny {
			return
		}
		if err := approve.SaveFileGrant(policyPath, grant, write); err != nil {
			// Loud on stderr (the CLI channel): the user approved, but the
			// grant is not in effect. The run continues without it.
			fmt.Fprintf(os.Stderr, "warden approval: could not save filesystem grant %q: %v; continuing without it\n", grant, err)
			_ = logger.Log(audit.Event{Type: "approval", Action: "deny", Resource: grant, Allowed: false, Reason: fmt.Sprintf("approved grant could not be saved: %v", err)})
			return
		}
		if freshPol, err := policy.Load(policyPath); err == nil {
			mu.Lock()
			snapshot = freshPol
			mu.Unlock()
		}
		fmt.Fprintf(os.Stderr, "warden approval: saved filesystem grant %q to %s\n", grant, policyPath)
		if outcome == approve.FileRestart {
			requestRestart()
		}
	}
	if err := approve.TailTraceFile(ctx, tracePath, emit); err != nil && ctx.Err() == nil {
		_ = logger.Log(audit.Event{Type: "approval", Action: "unavailable", Resource: "filesystem", Allowed: false, Reason: fmt.Sprintf("approval watch ended: %v", err)})
	}
}
