//go:build linux

package linux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/warden-sandbox/warden/internal/policy"
)

// LimitExceededError reports an enforced policy limit. Its explicit type
// lets the CLI distinguish a server failure from an intentional termination.
type LimitExceededError struct {
	Kind  string
	Limit string
}

func (e *LimitExceededError) Error() string {
	return fmt.Sprintf("resource limit exceeded: %s (%s)", e.Kind, e.Limit)
}

// waitWithLimits waits for cmd while monitoring the whole child process tree.
// Commands run in their own process group, so graceful termination reaches
// the bridge, server, and any workers rather than only the outer launcher.
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
	// A negative PID signals the process group. Ignore ESRCH: the command may
	// have exited in the small race between monitoring and termination.
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	select {
	case err := <-done:
		return err
	case <-time.After(750 * time.Millisecond):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		return <-done
	}
}

func processTreeRSS(root int) (uint64, error) {
	seen := make(map[int]bool)
	queue := []int{root}
	var total uint64
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		if pid <= 0 || seen[pid] {
			continue
		}
		seen[pid] = true
		rss, err := processRSS(pid)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return 0, err
		}
		total += rss
		children, err := processChildren(pid)
		if err != nil && !os.IsNotExist(err) {
			return 0, err
		}
		queue = append(queue, children...)
	}
	return total, nil
}

func processRSS(pid int) (uint64, error) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "status"))
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, fmt.Errorf("invalid VmRSS for pid %d", pid)
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse VmRSS for pid %d: %w", pid, err)
		}
		return kb * 1024, nil
	}
	return 0, nil
}

func processChildren(pid int) ([]int, error) {
	paths, err := filepath.Glob(filepath.Join("/proc", strconv.Itoa(pid), "task", "*", "children"))
	if err != nil {
		return nil, err
	}
	var out []int
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, field := range strings.Fields(string(data)) {
			child, err := strconv.Atoi(field)
			if err == nil {
				out = append(out, child)
			}
		}
	}
	return out, nil
}
