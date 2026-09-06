//go:build windows

package sandbox

import (
	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/policy"
	windowsbackend "github.com/warden-sandbox/warden/internal/sandbox/windows"
)

func runWindows(cmd []string, p policy.Policy) (int, error) {
	return windowsbackend.Run(cmd, p)
}

func runWindowsWithApproval(cmd []string, p policy.Policy, cfg *approve.Config) (int, error) {
	return windowsbackend.RunWithApproval(cmd, p, cfg)
}
