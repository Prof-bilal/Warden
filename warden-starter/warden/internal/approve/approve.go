// Package approve implements interactive approval mode (ROADMAP M7): the
// first time a sandboxed server requests an access outside its policy, Warden
// prompts the user on the controlling terminal instead of only hard-failing.
//
// Two enforcement realities shape the design:
//
//   - Network grants CAN apply live: the egress proxy decides per request,
//     so an approved host takes effect immediately.
//   - Filesystem grants CANNOT apply live: bwrap bind mounts are fixed at
//     spawn. An approved path is saved to the policy file and takes effect
//     on restart (the backend offers to restart the server at once).
//
// Security invariants:
//
//   - Prompts go to the controlling terminal (/dev/tty), never to the
//     server's stdin/stdout/stderr — stdio belongs to the MCP client and
//     must stay protocol-clean.
//   - No terminal, unreadable answer, timeout, or save failure all resolve
//     to Deny (fail closed), and every decision is audit-logged.
//   - Approval never widens the policy silently: session grants live in
//     memory only, and file grants always persist (a restart reloads from
//     the file, so memory-only filesystem approvals would lie).
package approve

import (
	"errors"
	"time"
)

// ErrRestartRequested signals that the user approved a filesystem grant and
// asked for an immediate restart so the widened policy takes effect. The CLI
// reloads the policy file and respawns the server; backends return it after
// terminating the current sandboxed process.
var ErrRestartRequested = errors.New("approval granted for a filesystem grant; restart requested")

// Config carries the approval-mode settings from the CLI to the backends.
// The zero value disables approval; backends treat a nil *Config as
// plain deny-by-default enforcement.
type Config struct {
	// Enabled turns denials into terminal prompts.
	Enabled bool
	// PolicyPath is the policy file persistent approvals are written to.
	// Required when Enabled.
	PolicyPath string
	// Timeout bounds each prompt. Zero waits indefinitely; expiry denies.
	Timeout time.Duration
}

// Validate checks the config before any sandboxed process starts.
func (c *Config) Validate() error {
	if c == nil || !c.Enabled {
		return nil
	}
	if c.PolicyPath == "" {
		return errors.New("approval mode requires a policy file to persist grants to")
	}
	if c.Timeout < 0 {
		return errors.New("approval timeout must not be negative")
	}
	return nil
}

// CheckTTY fails closed when there is no interactive terminal to prompt on:
// without a human at a controlling terminal, approval mode could never
// approve anything, so refusing to start is louder and safer than silently
// denying every request for the whole run.
func CheckTTY() error {
	f, err := openConsole()
	if err != nil {
		return errors.New("approval mode needs an interactive terminal (/dev/tty) to prompt on; run without --approve or from a terminal")
	}
	return f.Close()
}
