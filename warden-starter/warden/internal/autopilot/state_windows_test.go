//go:build windows

package autopilot

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestAutopilotRejectsSharedEnrollmentACL(t *testing.T) {
	project := t.TempDir()
	dir := filepath.Join(t.TempDir(), "state")
	state, err := OpenState(dir, project, true)
	if err != nil {
		t.Fatal(err)
	}
	defer state.Close()
	program, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	approval, err := Enroll(project, program, ReadOnly)
	if err != nil {
		t.Fatal(err)
	}
	if err := Save(state, approval); err != nil {
		t.Fatal(err)
	}

	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.User.Sid.String() + ")(A;;FR;;;WD)")
	if err != nil {
		t.Fatal(err)
	}
	acl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, enrollmentName(project))
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(state, project); err == nil {
		t.Fatal("loaded enrollment with a shared ACL")
	}
}
