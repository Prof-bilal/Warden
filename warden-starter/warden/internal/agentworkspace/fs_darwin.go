//go:build darwin

package agentworkspace

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func validatePortableFile(_ *os.File, info os.FileInfo) error {
	if st, ok := info.Sys().(*syscall.Stat_t); ok && st.Nlink != 1 {
		return fmt.Errorf("hard-linked files are unsupported")
	}
	return nil
}

func openPortable(path string) (*os.File, error) {
	// O_NOFOLLOW prevents a final-component symlink from being opened. Parent
	// components are checked by the projection walker and the post-open stat.
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), filepath.Base(path)), nil
}

func writePortable(path string, data []byte, mode os.FileMode, replace bool) error {
	parent := filepath.Dir(path)
	if info, err := os.Stat(parent); err != nil || !info.IsDir() {
		return fmt.Errorf("parent must already exist")
	}
	tmp, err := os.CreateTemp(parent, ".warden-apply-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err = tmp.Chmod(mode.Perm()); err != nil {
		tmp.Close()
		return err
	}
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if !replace {
		if _, err = os.Lstat(path); err == nil {
			return fmt.Errorf("new file conflicts with host")
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if err = os.Rename(tmpName, path); err != nil {
		return err
	}
	return nil
}
