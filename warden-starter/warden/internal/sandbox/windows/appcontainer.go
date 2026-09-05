//go:build windows

package windows

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// This file implements the AppContainer (LowBox) token and the filesystem
// capabilities derived from the policy. Both must succeed before a process is
// started; a failure is a fail-closed gate, never a compromise.

// acl and securityDescriptor are opaque handles to Windows ACL / security
// descriptor buffers that the advapi32 ACL API Fill in and out; their internal
// layout is owned by the OS and never read from Go.
type acl struct{}
type securityDescriptor struct{}

// trustee mirrors TRUSTEE_W (accctrl.h).
type trustee struct {
	multipleTrustee          *trustee
	multipleTrusteeOperation int32
	trusteeForm              uint32
	trusteeType              uint32
	ptstrName                *uint16 // for TRUSTEE_IS_SID this holds the SID bytes
}

// explicitAccess mirrors EXPLICIT_ACCESS_W (accctrl.h).
type explicitAccess struct {
	grfAccessPermissions uint32
	grfAccessMode        uint32 // ACCESS_MODE
	grfInheritance       uint32
	trustee              trustee
}

// Security-information and object-type constants (winnt.h).
const (
	seFileObject             = 1 // SE_FILE_OBJECT
	daclSecurityInformation  = 4 // DACL_SECURITY_INFORMATION
	protectedDaclInformation = 0x80000000
)

// Inheritance flags for the granted ACE (winnt.h) so children of a granted
// directory inherit the grant.
const (
	objectInheritAce    = 0x00000001
	containerInheritAce = 0x00000002
)

// Trustee forms (accctrl.h).
const trusteeIsSid = 3 // TRUSTEE_IS_SID

// _GetNamedSecurityInfoW returns the current security descriptor for a path.
func _GetNamedSecurityInfoW(path *uint16, objectType, securityInfo uint32, owner, group **windows.SID, dacl, sacl **acl, sd **securityDescriptor) error {
	r, _, _ := procGetNamedSecurityInfoW.Call(
		uintptr(unsafe.Pointer(path)),
		uintptr(objectType),
		uintptr(securityInfo),
		uintptr(unsafe.Pointer(owner)),
		uintptr(unsafe.Pointer(group)),
		uintptr(unsafe.Pointer(dacl)),
		uintptr(unsafe.Pointer(sacl)),
		uintptr(unsafe.Pointer(sd)),
	)
	if r != 0 {
		return fmt.Errorf("GetNamedSecurityInfoW: %w", windows.Errno(r))
	}
	return nil
}

// _SetNamedSecurityInfoW applies a new DACL to a path. A nil pointer leaves the
// corresponding field unchanged.
func _SetNamedSecurityInfoW(path *uint16, objectType, securityInfo uint32, owner, group *windows.SID, dacl, sacl *acl) error {
	r, _, _ := procSetNamedSecurityInfoW.Call(
		uintptr(unsafe.Pointer(path)),
		uintptr(objectType),
		uintptr(securityInfo),
		uintptr(unsafe.Pointer(owner)),
		uintptr(unsafe.Pointer(group)),
		uintptr(unsafe.Pointer(dacl)),
		uintptr(unsafe.Pointer(sacl)),
	)
	if r != 0 {
		return fmt.Errorf("SetNamedSecurityInfoW: %w", windows.Errno(r))
	}
	return nil
}

// _SetEntriesInAclW merges explicit grant entries into an existing ACL and
// returns a freshly allocated ACL the caller must LocalFree.
func _SetEntriesInAclW(entries []explicitAccess, oldAcl *acl, newAcl **acl) error {
	var first *explicitAccess
	if len(entries) > 0 {
		first = &entries[0]
	}
	r, _, _ := procSetEntriesInAclW.Call(
		uintptr(len(entries)),
		uintptr(unsafe.Pointer(first)),
		uintptr(unsafe.Pointer(oldAcl)),
		uintptr(unsafe.Pointer(newAcl)),
	)
	if r != 0 {
		return fmt.Errorf("SetEntriesInAcl: %w", windows.Errno(r))
	}
	return nil
}

// makeLowBoxToken derives an AppContainer token from the current process's
// token, using the plan's package SID and derived capability SIDs. The caller
// must close the returned handle.
func makeLowBoxToken(plan *Plan) (windows.Handle, error) {
	var base windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), tokenAllAccess, &base); err != nil {
		return 0, failClose("open process token", err)
	}
	defer base.Close()

	pkg, err := windows.StringToSid(plan.PackageSID)
	if err != nil {
		return 0, failClose("parse package SID "+plan.PackageSID, err)
	}

	caps := make([]sidAndAttributes, 0, len(plan.CapabilitySIDs))
	for _, s := range plan.CapabilitySIDs {
		sid, err := windows.StringToSid(s)
		if err != nil {
			return 0, failClose("parse capability SID "+s, err)
		}
		caps = append(caps, sidAndAttributes{Sid: sid, Attributes: capabilityAttribute})
	}

	// NtCreateLowBoxToken accepts an existing primary token handle; the process
	// token we opened qualifies directly, so no duplicate is needed.
	var token windows.Handle
	if err := _NtCreateLowBoxToken(&token, windows.Handle(base), tokenAllAccess, pkg, caps); err != nil {
		return 0, failClose("create AppContainer token", err)
	}
	return token, nil
}

// planCapabilitySIDs resolves every capability SID the plan derives for the
// run into binary SIDs used for both the token and the filesystem ACEs.
func planCapabilitySIDs(plan *Plan) ([]*windows.SID, error) {
	out := make([]*windows.SID, 0, len(plan.CapabilitySIDs))
	for _, s := range plan.CapabilitySIDs {
		sid, err := windows.StringToSid(s)
		if err != nil {
			return nil, fmt.Errorf("parse capability SID %q: %w", s, err)
		}
		out = append(out, sid)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("plan carries no capability SID for filesystem grants")
	}
	return out, nil
}

// freeACL releases an ACL or security descriptor returned by advapi32.
func freeACL(p unsafe.Pointer) {
	if p == nil {
		return
	}
	_, _ = windows.LocalFree(windows.Handle(p))
}

// grantFilesystemAccess appends the policy-derived capability ACEs to each
// granted path's DACL. Because AppContainer tokens carry none of the default
// group SIDs (INTERACTIVE/USERS/Everyone), the existing DACLs do not help the
// sandbox by themselves; adding the capability grant is necessary and safe,
// and appending to the current DACL never strips the caller's own rights.
//
// Runtime read paths (system locations) degrade gracefully: if the OS already
// grants the AppContainer read access (the standard ALL APPLICATION PACKAGES
// ACE), a failed grant there is tolerated; any failure on a policy path fails
// the run closed rather than weakening the sandbox.
func grantFilesystemAccess(plan *Plan) error {
	caps, err := planCapabilitySIDs(plan)
	if err != nil {
		return err
	}
	for _, g := range plan.Grants {
		if err := grantPathACE(g.Path, g.AccessMask, caps); err != nil {
			if IsRuntimeReadPath(g.Path) {
				continue
			}
			return fmt.Errorf("grant filesystem access to %q: %w", g.Path, err)
		}
	}
	return nil
}

// grantPathACE appends one GRANT_ACCESS ACE per capability SID to path's DACL
// with the grant's access mask, then writes the merged DACL back.
func grantPathACE(path string, accessMask uint32, caps []*windows.SID) error {
	pathPtr := utf16Ptr(path)

	var dacl *acl
	var sd *securityDescriptor
	if err := _GetNamedSecurityInfoW(pathPtr, seFileObject, daclSecurityInformation, nil, nil, &dacl, nil, &sd); err != nil {
		return err
	}
	if sd != nil {
		defer freeACL(unsafe.Pointer(sd))
	}

	entries := make([]explicitAccess, 0, len(caps))
	for _, sid := range caps {
		entries = append(entries, explicitAccess{
			grfAccessPermissions: accessMask,
			grfAccessMode:        grantAccess,
			grfInheritance:       objectInheritAce | containerInheritAce,
			trustee:              trustee{trusteeForm: trusteeIsSid, ptstrName: (*uint16)(unsafe.Pointer(sid))},
		})
	}

	var newDacl *acl
	if err := _SetEntriesInAclW(entries, dacl, &newDacl); err != nil {
		return err
	}
	if newDacl != nil {
		defer freeACL(unsafe.Pointer(newDacl))
	}

	protect := uint32(protectedDaclInformation) | daclSecurityInformation
	return _SetNamedSecurityInfoW(pathPtr, seFileObject, protect, nil, nil, newDacl, nil)
}
