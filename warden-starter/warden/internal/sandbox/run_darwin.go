//go:build darwin

package sandbox

import (
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/sandbox/darwin"
)

func runSeatbelt(cmd []string, p policy.Policy) (int, error) {
	return darwin.Run(cmd, p)
}
