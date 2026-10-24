//go:build windows

package windows

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Runtimes resolve real paths by stat-ing every component from the drive
// root down (node's fs.realpathSync, Python's os.path.realpath, ...). An
// AppContainer has no access to C:\ or to the user's folders above a grant,
// so those lstat calls fail with EPERM before the target even loads its
// entry script.
//
// grantAncestorAttributes gives the run's capability SID
// SYNCHRONIZE|FILE_READ_ATTRIBUTES on each ancestor directory of the grants,
// on that directory only (no inheritance, no listing, no content access).
// The target learns only that the ancestors exist and their attributes,
// which the granted paths themselves already reveal.
//
// It uses SetFileSecurity rather than SetNamedSecurityInfo on purpose:
// SetNamedSecurityInfo re-propagates inheritance through the whole subtree,
// which on C:\ would rewrite every ACL on the disk. SetFileSecurity changes
// exactly one object. The original descriptors are written back by the
// returned restore func. Failures are best effort (a non-administrator cannot
// change C:\): the run continues with the stricter view.
const ancestorAccess = windows.SYNCHRONIZE | 0x0080 // FILE_READ_ATTRIBUTES

var procSetFileSecurityW = windows.NewLazySystemDLL("advapi32.dll").NewProc("SetFileSecurityW")

func setFileSecurity(path string, sd *windows.SECURITY_DESCRIPTOR) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	r, _, e := procSetFileSecurityW.Call(uintptr(unsafe.Pointer(p)), uintptr(windows.DACL_SECURITY_INFORMATION), uintptr(unsafe.Pointer(sd)))
	if r == 0 {
		return e
	}
	return nil
}

// ancestorDirs returns every ancestor directory of the grant paths, drive
// root included, that is not itself a grant, deduplicated case-insensitively.
func ancestorDirs(grants []pathGrant) []string {
	granted := map[string]bool{}
	for _, g := range grants {
		granted[strings.ToLower(cleanWinPath(g.Path))] = true
	}
	seen := map[string]bool{}
	var out []string
	for _, g := range grants {
		dir := cleanWinPath(g.Path)
		for !isDriveRootWin(dir) {
			dir = parentDirWin(dir)
			key := strings.ToLower(dir)
			if seen[key] || granted[key] {
				continue
			}
			seen[key] = true
			out = append(out, dir)
		}
	}
	return out
}

func grantAncestorAttributes(plan *Plan) func() {
	caps, err := planCapabilitySIDs(plan)
	if err != nil || len(caps) == 0 {
		return func() {}
	}
	type saved struct {
		path string
		sd   *windows.SECURITY_DESCRIPTOR
	}
	var restore []saved
	for _, dir := range ancestorDirs(plan.Grants) {
		path := longPath(dir)
		orig, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			continue
		}
		dacl, _, err := orig.DACL()
		if err != nil {
			continue
		}
		control, _, err := orig.Control()
		if err != nil {
			continue
		}
		entries := make([]windows.EXPLICIT_ACCESS, 0, len(caps))
		for _, sid := range caps {
			entries = append(entries, windows.EXPLICIT_ACCESS{
				AccessPermissions: ancestorAccess,
				AccessMode:        windows.GRANT_ACCESS,
				Inheritance:       windows.NO_INHERITANCE,
				Trustee: windows.TRUSTEE{
					TrusteeForm:  windows.TRUSTEE_IS_SID,
					TrusteeType:  windows.TRUSTEE_IS_UNKNOWN,
					TrusteeValue: windows.TrusteeValueFromSID(sid),
				},
			})
		}
		newDACL, err := windows.ACLFromEntries(entries, dacl)
		if err != nil {
			continue
		}
		sd, err := windows.NewSecurityDescriptor()
		if err != nil {
			continue
		}
		if err := sd.SetDACL(newDACL, true, false); err != nil {
			continue
		}
		if err := preserveDACLControl(sd, control); err != nil {
			continue
		}
		if err := setFileSecurity(path, sd); err != nil {
			continue // not ours to change (e.g. C:\ without elevation)
		}
		if err := preserveDACLControl(orig, control); err != nil {
			continue
		}
		restore = append(restore, saved{path: path, sd: orig})
	}
	return func() {
		for i := len(restore) - 1; i >= 0; i-- {
			_ = setFileSecurity(restore[i].path, restore[i].sd)
		}
	}
}

// preserveDACLControl carries the original protected/auto-inherited DACL
// state onto sd. SetFileSecurity drops SE_DACL_AUTO_INHERITED unless
// SE_DACL_AUTO_INHERIT_REQ is set too, which silently turned C:'s "PAI"
// DACL into "P".
func preserveDACLControl(sd *windows.SECURITY_DESCRIPTOR, control windows.SECURITY_DESCRIPTOR_CONTROL) error {
	mask := windows.SECURITY_DESCRIPTOR_CONTROL(windows.SE_DACL_PROTECTED | windows.SE_DACL_AUTO_INHERITED | windows.SE_DACL_AUTO_INHERIT_REQ)
	bits := control & (windows.SE_DACL_PROTECTED | windows.SE_DACL_AUTO_INHERITED)
	if control&windows.SE_DACL_AUTO_INHERITED != 0 {
		bits |= windows.SE_DACL_AUTO_INHERIT_REQ
	}
	return sd.SetControl(mask, bits)
}
