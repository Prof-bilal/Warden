//go:build !linux

package docker

import "os"

// hostUIDGID returns the invoking user's uid/gid on platforms that do not
// have Linux user namespaces. Windows has no uid/gid concept
// (os.Getuid/os.Getgid return -1), so the caller omits --user there.
func hostUIDGID() (int, int, bool) {
	uid, gid := os.Getuid(), os.Getgid()
	return uid, gid, uid >= 0 && gid >= 0
}
