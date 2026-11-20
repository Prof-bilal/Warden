package sandbox

import (
	"fmt"
	"time"

	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/sandbox/docker"
)

// RunWithApproval executes cmd like Run with interactive approval mode
// (M7). A nil or disabled cfg behaves exactly like Run. On Linux, blocked
// network requests prompt and apply live while blocked file accesses prompt
// with an offer to save and restart (backends return approve.ErrRestartRequested
// after terminating the run so the CLI can respawn under the widened policy).
// On other backends only network approval is livethere is no comparable
// live filesystem signaland filesystem stays hard-deny.
func RunWithApproval(cmd []string, p policy.Policy, backend string, cfg *approve.Config) (int, error) {
	if p.Agent != nil {
		return 0, fmt.Errorf("agent policy requires `warden agent`")
	}
	if cfg == nil || !cfg.Enabled {
		return Run(cmd, p, backend)
	}
	if err := cfg.Validate(); err != nil {
		return 0, err
	}
	name, err := Resolve(backend)
	if err != nil {
		return 0, err
	}
	p = p.WithActiveGrants(time.Now())
	if err := checkGrantLifetime(p, name); err != nil {
		return 0, err
	}
	switch name {
	case BackendLinux:
		return runLinuxWithApproval(cmd, p, cfg)
	case BackendSeatbelt:
		return runSeatbeltWithApproval(cmd, p, cfg)
	case BackendWindows:
		return runWindowsWithApproval(cmd, p, cfg)
	case BackendDocker:
		return docker.RunWithApproval(cmd, p, cfg)
	default:
		return 0, fmt.Errorf("internal error: unresolved backend %q", name)
	}
}
