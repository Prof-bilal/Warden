// Package sandboxerr defines shared error types for sandbox backend
// initialization failures. Both the parent sandbox package and platform-specific
// backends (windows, linux, darwin, docker) can import this package without
// creating import cycles.
package sandboxerr

import "fmt"

// RefuseToRun is returned when Warden cannot start the sandbox backend and
// must refuse to run the target command. It carries the specific reason so
// the CLI can present a clear, actionable message.
type RefuseToRun struct {
	Reason string
}

func (e RefuseToRun) Error() string {
	return fmt.Sprintf(`✗ Warden refused to start: sandbox backend unavailable

The sandbox backend couldn't initialize because %s.

This is not a bug — Warden fails closed by design. It will never run
your MCP server without a working sandbox, since that would mean
running it completely unprotected.

To fix this:
  → Re-run this command from an elevated (Administrator) terminal

Warden never falls back to running unsandboxed, under any circumstances.`, e.Reason)
}
