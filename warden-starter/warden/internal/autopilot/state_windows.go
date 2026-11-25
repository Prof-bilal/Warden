//go:build windows

package autopilot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func createStateDirectory(dir string) error {
	if err := os.MkdirAll(filepath.Dir(dir), 0700); err != nil {
		return err
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	sid := user.User.Sid.String()
	sd, err := windows.SecurityDescriptorFromString("O:" + sid + "D:P(A;OICI;FA;;;" + sid + ")")
	if err != nil {
		return err
	}
	path, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return err
	}
	attrs := windows.SecurityAttributes{SecurityDescriptor: sd}
	attrs.Length = uint32(unsafe.Sizeof(attrs))
	if err := windows.CreateDirectory(path, &attrs); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	return nil
}

func validStateEntry(info os.FileInfo) bool {
	attrs, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return ok && attrs.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0
}

func privateStateDirectory(state *os.Root, _ os.FileInfo) bool {
	dir, err := state.Open(".")
	if err != nil {
		return false
	}
	defer dir.Close()
	return privateStateHandle(dir, true)
}

func openEnrollment(state *os.Root, name string) (*os.File, error) {
	before, err := state.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || !validStateEntry(before) {
		return nil, fmt.Errorf("invalid Autopilot enrollment file")
	}
	f, err := state.Open(name)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || !os.SameFile(before, info) {
		f.Close()
		return nil, fmt.Errorf("Autopilot enrollment changed during open")
	}
	return f, nil
}

func privateEnrollment(f *os.File, _ os.FileInfo) bool {
	return privateStateHandle(f, false)
}

func privateStateHandle(f *os.File, directory bool) bool {
	handle := windows.Handle(f.Fd())
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil || info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return false
	}
	sd, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return false
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil {
		return false
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil || !owner.Equals(user.User.Sid) && (directory || !owner.IsWellKnown(windows.WinBuiltinAdministratorsSid)) {
		return false
	}
	_, dacl, ok := strings.Cut(sd.String(), "D:")
	if !ok {
		return false
	}
	_, ace, ok := strings.Cut(dacl, "(")
	if !ok {
		return false
	}
	// Comparing the single ACE also rejects additional allow or inherited ACEs.
	sid := user.User.Sid.String()
	if directory {
		control, _, err := sd.Control()
		return err == nil && control&windows.SE_DACL_PROTECTED != 0 && ace == "A;OICI;FA;;;"+sid+")"
	}
	return ace == "A;ID;FA;;;"+sid+")" || ace == "A;;FA;;;"+sid+")"
}
