//go:build !linux

package sandbox

import (
	"fmt"
	"runtime"

	"github.com/warden-sandbox/warden/internal/policy"
)

// Run fails closed until a backend can enforce every policy dimension on the
// current platform. In particular, it never falls back to executing cmd
// directly with the user's ambient permissions.
func Run(_ []string, _ policy.Policy) (int, error) {
	return 0, fmt.Errorf("%s sandbox backend is not implemented; refusing to run without filesystem, network, environment, audit, and limit enforcement", runtime.GOOS)
}
