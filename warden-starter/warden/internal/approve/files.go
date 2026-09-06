package approve

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/policy"
)

// SaveHostGrant reloads the policy file fresh, adds a network host, and
// saves it back. Reloading (instead of patching a stale in-memory copy)
// keeps concurrent approvals — e.g. a network and a filesystem prompt in
// the same run — from clobbering each other's grants.
func SaveHostGrant(policyPath, host string) error {
	p, err := policy.Load(policyPath)
	if err != nil {
		return fmt.Errorf("reload policy for approval: %w", err)
	}
	if _, err := p.AddHost(host); err != nil {
		return fmt.Errorf("add network grant: %w", err)
	}
	if err := p.Save(policyPath); err != nil {
		return fmt.Errorf("save approved grant: %w", err)
	}
	return nil
}

// SaveFileGrant reloads the policy file fresh, adds a filesystem grant, and
// saves it back. Write grants inside the runtime base (/usr, /lib64, ...)
// are refused: no backend could honor them, so approving would be a lie.
func SaveFileGrant(policyPath, grant string, write bool) error {
	if write {
		for _, base := range []string{"/usr", "/lib", "/lib64", "/bin", "/proc", "/dev", "/sys"} {
			if grant == base || policy.Within(grant, base) {
				return fmt.Errorf("grant %q is inside the read-only runtime base %s and cannot be granted write", grant, base)
			}
		}
	}
	p, err := policy.Load(policyPath)
	if err != nil {
		return fmt.Errorf("reload policy for approval: %w", err)
	}
	if _, err := p.AddFileGrant(grant, write); err != nil {
		return fmt.Errorf("add filesystem grant: %w", err)
	}
	if err := p.Save(policyPath); err != nil {
		return fmt.Errorf("save approved grant: %w", err)
	}
	return nil
}

// ShouldPromptFile decides whether a failed file syscall deserves a prompt.
// Only genuine sandbox denials qualify: the access must map to a grant
// (absolute, non-runtime path), fall outside the current policy, and name a
// path that exists on the host — otherwise the event is either already
// allowed, ungrantable, or a routine probe for an optional file, and
// prompting would train the user to click through noise. cmdExe (the
// server's own executable, whose parent dir backends bind read-only) is
// excluded.
func ShouldPromptFile(ev audit.Event, pol *policy.Policy, cmdExe string) (grant string, write bool, ok bool) {
	if ev.Type != "file" || ev.Allowed {
		return "", false, false
	}
	grant, write, ok = policy.ProposeFileGrant(ev.Action, ev.Resource)
	if !ok {
		return "", false, false
	}
	if grant == cmdExe || policy.CoversFile(*pol, grant) {
		return "", false, false
	}
	if _, err := os.Stat(grant); err != nil {
		return "", false, false
	}
	// A read grant on a directory the server merely listed is fine; a write
	// grant needs an existing directory (bwrap/docker --bind requires one).
	if write {
		if info, err := os.Stat(grant); err != nil || !info.IsDir() {
			return "", false, false
		}
	}
	return grant, write, true
}

// TailTraceFile follows a live strace log the way `warden logs --follow`
// follows the audit log: it polls for new bytes, parses only complete lines
// (a trailing partial line is held for the next poll), and emits each
// auditable event. Parsing is identical to batch import
// (audit.ParseStraceLine). It returns when ctx ends; a missing file is an
// error so the caller can warn loudly instead of silently offering no
// filesystem approvals.
func TailTraceFile(ctx context.Context, path string, emit func(audit.Event)) error {
	var offset int64
	var pending string
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}
		data, size, err := readNewBytes(path, &offset)
		if err != nil {
			return err
		}
		_ = size
		if len(data) == 0 {
			continue
		}
		pending += string(data)
		lastNL := -1
		for i := len(pending) - 1; i >= 0; i-- {
			if pending[i] == '\n' {
				lastNL = i
				break
			}
		}
		if lastNL < 0 {
			continue
		}
		complete, rest := pending[:lastNL], pending[lastNL+1:]
		pending = rest
		for _, line := range splitLines(complete) {
			if ev, ok := audit.ParseStraceLine(line); ok {
				emit(ev)
			}
		}
	}
}

func readNewBytes(path string, offset *int64) ([]byte, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("open trace for approval watch: %w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("stat trace for approval watch: %w", err)
	}
	size := info.Size()
	if size < *offset {
		*offset = 0 // rotation/truncation: reread from the start
	}
	if size == *offset {
		return nil, size, nil
	}
	if _, err := f.Seek(*offset, 0); err != nil {
		return nil, size, fmt.Errorf("seek trace for approval watch: %w", err)
	}
	buf := make([]byte, size-*offset)
	n := 0
	for n < len(buf) {
		m, err := f.Read(buf[n:])
		n += m
		if err != nil {
			break
		}
	}
	*offset += int64(n)
	return buf[:n], size, nil
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
