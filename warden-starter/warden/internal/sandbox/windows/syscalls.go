//go:build windows

package windows

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// This file binds the Win32 API surface the Warden Windows backend needs that
// golang.org/x/sys/windows does not provide. It deliberately uses LazyProc
// handles rather than mkwinsyscall //sys directives so the package builds with
// an unmodified toolchain. Struct mirrors follow the SDK headers; every mirror
// that crosses the enforcement boundary is flagged with its header and must be
// re-verified when CI gains a Windows runner.
//
// Component structs (ACL/ACE in appcontainer.go, FWPM_* in wfp.go,
// EVENT_TRACE_* in etw.go, JOBOBJECT_* in job.go) live with their code; this
// file owns the shared LazyDLL handles, proc pointers, and tiny wrappers.

var (
	ntdll    = windows.NewLazySystemDLL("ntdll.dll")
	advapi32 = windows.NewLazySystemDLL("advapi32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	fwpuclnt = windows.NewLazySystemDLL("fwpuclnt.dll")

	// ntdll
	procNtCreateLowBoxToken = ntdll.NewProc("NtCreateLowBoxToken")

	// advapi32
	procCreateProcessWithTokenW = advapi32.NewProc("CreateProcessWithTokenW")
	procGetNamedSecurityInfoW   = advapi32.NewProc("GetNamedSecurityInfoW")
	procSetNamedSecurityInfoW   = advapi32.NewProc("SetNamedSecurityInfoW")
	procSetEntriesInAclW        = advapi32.NewProc("SetEntriesInAclW")
	procStartTraceW             = advapi32.NewProc("StartTraceW")
	procControlTraceW           = advapi32.NewProc("ControlTraceW")
	procEnableTraceEx2          = advapi32.NewProc("EnableTraceEx2")

	// kernel32
	procOpenTraceW               = kernel32.NewProc("OpenTraceW")
	procProcessTrace             = kernel32.NewProc("ProcessTrace")
	procCloseTrace               = kernel32.NewProc("CloseTrace")
	procQueryDosDeviceW          = kernel32.NewProc("QueryDosDeviceW")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = kernel32.NewProc("Process32FirstW")
	procProcess32Next            = kernel32.NewProc("Process32NextW")

	// fwpuclnt (Windows Filtering Platform user-mode API)
	procFwpmEngineOpen        = fwpuclnt.NewProc("FwpmEngineOpen")
	procFwpmEngineClose       = fwpuclnt.NewProc("FwpmEngineClose")
	procFwpmTransactionBegin  = fwpuclnt.NewProc("FwpmTransactionBegin")
	procFwpmTransactionCommit = fwpuclnt.NewProc("FwpmTransactionCommit")
	procFwpmTransactionAbort  = fwpuclnt.NewProc("FwpmTransactionAbort")
	procFwpmSublayerAdd       = fwpuclnt.NewProc("FwpmSublayerAdd")
	procFwpmFilterAdd         = fwpuclnt.NewProc("FwpmFilterAdd")
	procFwpmFreeMemory        = fwpuclnt.NewProc("FwpmFreeMemory")
	procFwpmFilterDeleteById  = fwpuclnt.NewProc("FwpmFilterDeleteById")
)

// lasterr maps a Windows BOOL-style Proc.Call error to a Go error, normalising
// the zero Errno so helpers can return it directly.
func lasterr(err error) error {
	if errno, ok := err.(syscall.Errno); ok && errno == 0 {
		return nil
	}
	return err
}

// ntStatusToErr converts a non-success NTSTATUS into a Go error.
func ntStatusToErr(status int32) error {
	return syscall.Errno(uint32(status))
}

// Access flags requested from NtCreateLowBoxToken.
const (
	tokenAssignPrimary   = 0x00000001
	tokenDuplicate       = 0x00000002
	tokenQuery           = 0x00000008
	tokenAdjustDefault   = 0x00000080
	tokenAdjustSessionID = 0x00000100
	tokenAllAccess       = 0x000F01FF
)

// sidAndAttributes mirrors SID_AND_ATTRIBUTES (winnt.h) for the capability
// list passed to NtCreateLowBoxToken.
type sidAndAttributes struct {
	Sid        *windows.SID
	Attributes uint32
}

// Capability attribute flags for a LowBox token. The capabilities a sandbox
// needs (its capability SIDs and the AppContainer-family SIDs) are marked
// enabled; SE_GROUP_ENABLED_BY_DEFAULT is required for them to be enforced.
const (
	seGroupEnabledByDefault uint32 = 0x00000002
	seGroupEnabled          uint32 = 0x00000004
	capabilityAttribute            = seGroupEnabledByDefault | seGroupEnabled
)

// _NtCreateLowBoxToken creates an AppContainer (LowBox) token from an existing
// primary token and a set of capability SIDs. Signature (ntddll.h / wnode):
//
//	NTSTATUS NtCreateLowBoxToken(PHANDLE token, HANDLE existing,
//	                            ACCESS_MASK desiredAccess,
//	                            POBJECT_ATTRIBUTES objectAttributes,
//	                            PSID appContainerSid, ULONG capabilityCount,
//	                            PSID_AND_ATTRIBUTES capabilities,
//	                            ULONG handleCount, PHANDLE *handles);
//
// objectAttributes and the extra handle list are NULL here. The caller closes
// the returned token.
func _NtCreateLowBoxToken(token *windows.Handle, existing windows.Handle, desiredAccess uint32, appContainerSid *windows.SID, caps []sidAndAttributes) error {
	status, _, _ := procNtCreateLowBoxToken.Call(
		uintptr(unsafe.Pointer(token)),
		uintptr(existing),
		uintptr(desiredAccess),
		0, // objectAttributes
		uintptr(unsafe.Pointer(appContainerSid)),
		uintptr(len(caps)),
		uintptr(unsafe.Pointer(&caps[0])),
		0, // handleCount
		0, // handles
	)
	if status >= 0 {
		return nil
	}
	return ntStatusToErr(int32(status))
}

// _CreateProcessWithTokenW runs an executable under a primary token with an
// explicit environment block and inherited standard handles (advapi32:
// CreateProcessWithTokenW).
func _CreateProcessWithTokenW(token windows.Handle, creationFlags uint32, cmdLine *uint16, env *uint16, curDir *uint16, si *startupInfo, pi *processInformation) error {
	r, _, e := procCreateProcessWithTokenW.Call(
		uintptr(token),
		0, // dwLogonFlags
		0, // lpApplicationName (resolved from the command line)
		uintptr(unsafe.Pointer(cmdLine)),
		uintptr(creationFlags),
		uintptr(unsafe.Pointer(env)),
		uintptr(unsafe.Pointer(curDir)),
		uintptr(unsafe.Pointer(si)),
		uintptr(unsafe.Pointer(pi)),
	)
	if r == 0 {
		return lasterr(e)
	}
	return nil
}

// _QueryDosDeviceW resolves a DOS drive (e.g. "C:") to its NT device target
// (kernel32.QueryDosDeviceW).
func _QueryDosDeviceW(deviceName *uint16, target *uint16, max uint32) (uint32, error) {
	r, _, e := procQueryDosDeviceW.Call(uintptr(unsafe.Pointer(deviceName)), uintptr(unsafe.Pointer(target)), uintptr(max))
	if r == 0 {
		return 0, lasterr(e)
	}
	return uint32(r), nil
}

// _CreateToolhelp32Snapshot opens a process snapshot (kernel32:
// CreateToolhelp32Snapshot). The INVALID_HANDLE_VALUE failure sentinel is
// translated to an error.
func _CreateToolhelp32Snapshot(flags, processID uint32) (windows.Handle, error) {
	r, _, e := procCreateToolhelp32Snapshot.Call(uintptr(flags), uintptr(processID))
	if r == ^uintptr(0) {
		return 0, lasterr(e)
	}
	return windows.Handle(r), nil
}

// _Process32First initializes a PROCESSENTRY32W from a snapshot.
func _Process32First(snapshot windows.Handle, entry *processEntry32W) error {
	r, _, e := procProcess32First.Call(uintptr(snapshot), uintptr(unsafe.Pointer(entry)))
	if r == 0 {
		return lasterr(e)
	}
	return nil
}

// _Process32Next advances to the next PROCESSENTRY32W in a snapshot.
func _Process32Next(snapshot windows.Handle, entry *processEntry32W) error {
	r, _, e := procProcess32Next.Call(uintptr(snapshot), uintptr(unsafe.Pointer(entry)))
	if r == 0 {
		return lasterr(e)
	}
	return nil
}
