//go:build windows

package sandbox

import (
	"github.com/warden-sandbox/warden/internal/policy"
	windowsbackend "github.com/warden-sandbox/warden/internal/sandbox/windows"
)

func runWindows(cmd []string, p policy.Policy) (int, error) {
	return windowsbackend.Run(cmd, p)
}
