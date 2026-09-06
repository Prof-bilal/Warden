//go:build !darwin

package sandbox

import (
	"fmt"
	"runtime"

	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/policy"
)

func runSeatbelt(_ []string, _ policy.Policy) (int, error) {
	return 0, fmt.Errorf("seatbelt sandbox backend is not available on %s", runtime.GOOS)
}

func runSeatbeltWithApproval(_ []string, _ policy.Policy, _ *approve.Config) (int, error) {
	return 0, fmt.Errorf("seatbelt sandbox backend is not available on %s", runtime.GOOS)
}
