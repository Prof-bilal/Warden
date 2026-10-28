//go:build windows

package agentworkspace

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procReplaceFileW = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReplaceFileW")

func validatePortableFile(f *os.File, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("reparse points are unsupported")
	}
	var data windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(f.Fd()), &data); err != nil {
		return err
	}
	if data.NumberOfLinks != 1 {
		return fmt.Errorf("hard-linked files are unsupported")
	}
	if data.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fmt.Errorf("reparse points are unsupported")
	}
	return nil
}

func openPortable(path string) (*os.File, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	// Opening with OPEN_REPARSE_POINT makes the final handle refer to the
	// directory entry itself, not a reparse target. This is the Windows
	// equivalent of O_NOFOLLOW for the projection reader.
	h, err := windows.CreateFile(p, windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(h), filepath.Base(path)), nil
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
	from, err := windows.UTF16PtrFromString(tmpName)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	if replace {
		// ReplaceFileW performs the replacement as one native operation and
		// preserves the target's ACLs and file identity. The replacement and
		// target are in the same temporary workspace parent by construction.
		r, _, callErr := procReplaceFileW.Call(
			uintptr(unsafe.Pointer(to)), uintptr(unsafe.Pointer(from)), 0, 0, 0, 0,
		)
		if r == 0 {
			if callErr != nil {
				return callErr
			}
			return fmt.Errorf("ReplaceFileW failed")
		}
		return nil
	} else if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("new file conflicts with host")
	} else if !os.IsNotExist(err) {
		return err
	}
	flags := uint32(windows.MOVEFILE_WRITE_THROUGH)
	if err = windows.MoveFileEx(from, to, flags); err != nil {
		return err
	}
	return nil
}
