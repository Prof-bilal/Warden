//go:build !windows

package sandbox

import (
	"fmt"
	"runtime"

	"github.com/warden-sandbox/warden/internal/policy"
)

func runWindows(_ []string, _ policy.Policy) (int, error) {
	return 0, fmt.Errorf("windows sandbox backend is not available on %s", runtime.GOOS)
}
