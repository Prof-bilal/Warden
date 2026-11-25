//go:build linux || darwin

package autopilot

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func createStateDirectory(dir string) error {
	return os.MkdirAll(dir, 0700)
}

func validStateEntry(info os.FileInfo) bool {
	return ownedPrivate(info, 0700)
}

func privateStateDirectory(_ *os.Root, info os.FileInfo) bool {
	return ownedPrivate(info, 0700)
}

func openEnrollment(state *os.Root, name string) (*os.File, error) {
	return state.OpenFile(name, os.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
}

func privateEnrollment(_ *os.File, info os.FileInfo) bool {
	return ownedPrivate(info, 0600)
}

func ownedPrivate(info os.FileInfo, mode os.FileMode) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid()) && info.Mode().Perm() == mode
}
