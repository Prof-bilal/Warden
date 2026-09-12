//go:build darwin

package darwin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/envfilter"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/proxy"
	"github.com/warden-sandbox/warden/internal/sandbox/sandboxerr"
)

// LimitExceededError reports an enforced policy limit.
type LimitExceededError struct {
	Kind  string
	Limit string
}

func (e *LimitExceededError) Error() string {
	return fmt.Sprintf("resource limit exceeded: %s (%s)", e.Kind, e.Limit)
}

// Run spawns cmd under sandbox-exec with a Seatbelt profile derived from p.
// Network egress is forced through the host-side proxy via a loopback bridge;
// Seatbelt denies all other network destinations. File denials are enforced
// by Seatbelt (EPERM); structured file-deny audit events are not available
// without a macOS tracing primitive comparable to Linux strace — network
// allow/deny decisions are still recorded by the egress proxy.
func Run(cmd []string, p policy.Policy) (int, error) {
	if _, err := exec.LookPath("sandbox-exec"); err != nil {
		return 0, sandboxerr.RefuseToRun{Reason: "sandbox-exec is not installed on this macOS host"}
	}
	logFile, _, err := audit.OpenDefault()
	if err != nil {
		return 0, fmt.Errorf("open audit log: %w", err)
	}
	defer logFile.Close()
	return runWithEnvAndAudit(cmd, p, os.Environ(), audit.New(logFile), nil)
}

// RunWithApproval spawns cmd like Run with interactive approval mode for
// network requests: blocked hosts prompt on the terminal and approved ones
// apply live via the egress proxy. Seatbelt has no Linux-strace-equivalent
// live file signal, so filesystem stays hard-deny (use `warden trace` +
// `warden init` to widen it). A nil or disabled cfg behaves like Run.
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
	sandboxExec, err := exec.LookPath("sandbox-exec")
	if err != nil {
		return 0, sandboxerr.RefuseToRun{Reason: "sandbox-exec is not installed on this macOS host"}
	}

	eg, err := proxy.Start(p.Network.Allow, logger)
	if err != nil {
		return 0, fmt.Errorf("start egress proxy: %w", err)
	}
	defer eg.Close()

	if approval != nil && approval.Enabled {
		eg.SetApprover(approve.NewPrompter(approval.PolicyPath, approval.Timeout, logger).NetworkApprover())
	}

	profile, err := BuildSeatbeltProfile(cmd, p, eg.SocketPath())
	if err != nil {
		return 0, fmt.Errorf("build seatbelt profile: %w", err)
	}
	profileFile, err := os.CreateTemp("", "warden-seatbelt-*.sb")
	if err != nil {
		return 0, fmt.Errorf("create seatbelt profile file: %w", err)
	}
	profilePath := profileFile.Name()
	defer os.Remove(profilePath)
	if _, err := profileFile.WriteString(profile); err != nil {
		profileFile.Close()
		return 0, fmt.Errorf("write seatbelt profile: %w", err)
	}
	if err := profileFile.Close(); err != nil {
		return 0, fmt.Errorf("close seatbelt profile: %w", err)
	}

	bridgeExe, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("locate proxy bridge executable: %w", err)
	}
	bridgeExe, err = filepath.EvalSymlinks(bridgeExe)
	if err != nil {
		return 0, fmt.Errorf("resolve proxy bridge executable: %w", err)
	}

	// sandbox-exec -f profile -- bridge __proxy-bridge ... -- cmd
	runArgs := []string{
		"-f", profilePath,
		bridgeExe, "__proxy-bridge",
		"--socket", eg.SocketPath(),
		"--listen", "127.0.0.1:18080",
		"--",
	}
	runArgs = append(runArgs, cmd...)

	sub := exec.Command(sandboxExec, runArgs...)
	sub.Stdin = os.Stdin
	sub.Stdout = os.Stdout
	sub.Stderr = os.Stderr
	// Start the sandboxed process in a directory that is always allowed
	// (TempDir) so shell `getcwd` does not fail with
	// "cannot access parent directories: Operation not permitted" when the
	// parent's cwd is the checkout under /Users which is deny-default.
	sub.Dir = os.TempDir()
	sub.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	sub.Env = append(envfilter.Filter(parentEnv, p.EnvAllowlist()),
		"HTTP_PROXY=http://127.0.0.1:18080",
		"HTTPS_PROXY=http://127.0.0.1:18080",
		"ALL_PROXY=http://127.0.0.1:18080",
		"NO_PROXY=",
	)

	if err := sub.Start(); err != nil {
		return 0, fmt.Errorf("start sandboxed process: %w", err)
	}
	runErr, limitErr := waitWithLimits(sub, p.Limits)
	if limitErr != nil {
		if logger != nil {
			_ = logger.Log(audit.Event{Type: "limit", Action: "terminate", Resource: limitErr.Limit, Allowed: false, Reason: limitErr.Error()})
		}
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

func waitWithLimits(cmd *exec.Cmd, limits policy.Limits) (error, *LimitExceededError) {
	if limits.MemoryMB == 0 && limits.TimeoutS == 0 {
		return cmd.Wait(), nil
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	var timeout <-chan time.Time
	var timer *time.Timer
	if limits.TimeoutS > 0 {
		timer = time.NewTimer(time.Duration(limits.TimeoutS) * time.Second)
		defer timer.Stop()
		timeout = timer.C
	}
	var ticks <-chan time.Time
	var ticker *time.Ticker
	if limits.MemoryMB > 0 {
		ticker = time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		ticks = ticker.C
	}
	maxBytes := uint64(limits.MemoryMB) * 1024 * 1024

	for {
		select {
		case err := <-done:
			return err, nil
		case <-timeout:
			err := terminateProcessGroup(cmd.Process.Pid, done)
			return err, &LimitExceededError{Kind: "wall-clock timeout", Limit: fmt.Sprintf("%ds", limits.TimeoutS)}
		case <-ticks:
			used, err := processTreeRSS(cmd.Process.Pid)
			if err == nil && used > maxBytes {
				err := terminateProcessGroup(cmd.Process.Pid, done)
				return err, &LimitExceededError{Kind: "memory", Limit: fmt.Sprintf("%dMB", limits.MemoryMB)}
			}
		}
	}
}

func terminateProcessGroup(pid int, done <-chan error) error {
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	select {
	case err := <-done:
		return err
	case <-time.After(750 * time.Millisecond):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		return <-done
	}
}

// processTreeRSS sums RSS for pid and descendants using `ps`, since macOS
// has no Linux-style /proc/<pid>/status.
func processTreeRSS(root int) (uint64, error) {
	out, err := exec.Command("ps", "-o", "rss=", "-g", strconv.Itoa(root)).Output()
	if err != nil {
		// Fall back to the single process if group listing fails.
		out, err = exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(root)).Output()
		if err != nil {
			return 0, err
		}
	}
	var total uint64
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		kb, err := strconv.ParseUint(line, 10, 64)
		if err != nil {
			continue
		}
		total += kb * 1024
	}
	return total, nil
}
