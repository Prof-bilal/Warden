//go:build linux

package sandbox

import (
	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/policy"
	linuxbackend "github.com/warden-sandbox/warden/internal/sandbox/linux"
)

func runLinux(cmd []string, p policy.Policy) (int, error) {
	return linuxbackend.Run(cmd, p)
}

func runLinuxWithApproval(cmd []string, p policy.Policy, cfg *approve.Config) (int, error) {
	return linuxbackend.RunWithApproval(cmd, p, cfg)
}
