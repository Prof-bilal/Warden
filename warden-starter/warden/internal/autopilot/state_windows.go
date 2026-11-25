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

func verifyStateDirectory(state *os.Root, info os.FileInfo) error {
	dir, err := os.Open(state.Name())
	if err != nil {
		return err
	}
	defer dir.Close()
	opened, err := dir.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return fmt.Errorf("state directory changed before ACL verification")
	}
	return verifyStateHandle(dir, true)
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
	return verifyStateHandle(f, false) == nil
}

func verifyStateHandle(f *os.File, directory bool) error {
	handle := windows.Handle(f.Fd())
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		return err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fmt.Errorf("reparse points are refused")
	}
	sd, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("read owner and DACL: %w", err)
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil {
		return fmt.Errorf("state owner is unavailable")
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	if !owner.Equals(user.User.Sid) && (directory || !owner.IsWellKnown(windows.WinBuiltinAdministratorsSid)) {
		return fmt.Errorf("unexpected state owner")
	}
	_, dacl, ok := strings.Cut(sd.String(), "D:")
	if !ok {
		return fmt.Errorf("owner-only DACL required")
	}
	_, ace, ok := strings.Cut(dacl, "(")
	if !ok {
		return fmt.Errorf("owner-only DACL required")
	}
	// SDDL may encode the current user as an alias such as LA.
	fields := strings.Split(strings.TrimSuffix(ace, ")"), ";")
	if len(fields) != 6 || fields[0] != "A" || fields[2] != "FA" || fields[3] != "" || fields[4] != "" {
		return fmt.Errorf("expected one full-access ACE for the current user")
	}
	trustee, err := windows.StringToSid(fields[5])
	if err != nil || !trustee.Equals(user.User.Sid) {
		return fmt.Errorf("state DACL does not grant access exclusively to the current user")
	}
	if directory {
		control, _, err := sd.Control()
		if err != nil || control&windows.SE_DACL_PROTECTED == 0 || fields[1] != "OICI" {
			return fmt.Errorf("state directory requires an inheritance-protected DACL with an inheritable user ACE")
		}
	} else if fields[1] != "ID" && fields[1] != "" {
		return fmt.Errorf("invalid enrollment ACE flags")
	}
	return nil
}
