//go:build !windows

// Package privatefile protects setup files before any secret-containing bytes
// are written. Windows needs ACLs, not Unix-style permission bits.
package privatefile

import "os"

func Protect(path string, directory bool) error {
	mode := os.FileMode(0600)
	if directory {
		mode = 0700
	}
	return os.Chmod(path, mode)
}
