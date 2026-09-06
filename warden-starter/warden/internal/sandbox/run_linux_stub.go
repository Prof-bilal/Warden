//go:build !linux

package sandbox

import (
	"fmt"
	"runtime"

	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/policy"
)

func runLinux(_ []string, _ policy.Policy) (int, error) {
	return 0, fmt.Errorf("linux sandbox backend is not available on %s", runtime.GOOS)
}

func runLinuxWithApproval(_ []string, _ policy.Policy, _ *approve.Config) (int, error) {
	return 0, fmt.Errorf("linux sandbox backend is not available on %s", runtime.GOOS)
}
