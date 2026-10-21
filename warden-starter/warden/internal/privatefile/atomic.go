package privatefile

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWrite replaces path with data without ever writing through the
// destination. It refuses symlinks and non-regular files, writes a protected
// same-directory temporary file, fsyncs it, and renames it into place. A crash
// therefore leaves either the old contents or the new contents, never a
// truncated file, and a symlinked destination cannot redirect the write to
// another file (for example ~/.ssh/authorized_keys).
func AtomicWrite(path string, data []byte) error {
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("refusing non-regular destination %q", path)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".warden-tmp-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = Protect(name, false); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
