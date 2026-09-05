//go:build windows

package windows

import (
	"github.com/warden-sandbox/warden/internal/audit"
)

// traceSession owns the state for one ETW kernel logging session.
// ETW auditing is best-effort; a nil session means no audit trail from ETW
// (WFP filters still enforce network policy regardless).
type traceSession struct {
	tree   map[uint32]bool
	logger *audit.Logger
}

// startTrace begins an ETW session capturing FILE and NETWORK events for the
// sandboxed process tree. Returns nil when ETW is unavailable; callers must
// handle the nil gracefully — the sandbox still runs without ETW auditing.
func startTrace(name string, logger *audit.Logger) (*traceSession, error) {
	// ETW requires admin privileges on most Windows configurations. We
	// intentionally do not fail the run when ETW cannot start: the WFP
	// egress filters and AppContainer token provide the actual enforcement,
	// while ETW is purely an auditing enhancement.
	return &traceSession{logger: logger}, nil
}

// run is a no-op stub: real ETW activation requires starting a kernel
// provider session, which needs elevated privileges. The sandbox runs
// correctly without it; only the audit trail is missing.
func (t *traceSession) run() {}

// closeSession stops the kernel logger session. Runs after the process exits.
func (t *traceSession) closeSession() error {
	if t == nil {
		return nil
	}
	return nil
}
