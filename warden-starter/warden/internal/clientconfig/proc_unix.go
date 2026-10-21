//go:build !windows

package clientconfig

import (
	"errors"
	"syscall"
)

// processAlive reports whether pid still exists. EPERM means the process
// exists but belongs to another user, which still counts as alive.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
