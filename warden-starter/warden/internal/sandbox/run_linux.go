//go:build linux

package sandbox

import (
	"github.com/warden-sandbox/warden/internal/policy"
	linuxbackend "github.com/warden-sandbox/warden/internal/sandbox/linux"
)

func runLinux(cmd []string, p policy.Policy) (int, error) {
	return linuxbackend.Run(cmd, p)
}
