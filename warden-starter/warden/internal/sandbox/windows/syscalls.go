//go:build windows

package windows

import (
	"errors"
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

	// wfpDLL is set by initWFP() — either fwpuclnt.dll (the canonical home of
	// the WFP user-mode API) or iphlapi.dll (which also exports the same
	// symbols on stripped Windows images, including the GitHub Actions
	// `windows-latest` runner which doesn't ship fwpuclnt.dll). Selecting at
	// init time means wfpSupported() can fail closed cleanly when neither
	// DLL is present, instead of panicking inside LazyProc.Call. See
	// https://learn.microsoft.com/en-us/windows/win32/fwp/ for the API.
	wfpDLL *windows.LazyDLL

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

	// kernel32 (process/enumeration helpers only; ETW consumer APIs are
	// advapi32, not kernel32 — bind them below with the other advapi32
	// procs so the whole session fails closed if any symbol is missing).
	procQueryDosDeviceW          = kernel32.NewProc("QueryDosDeviceW")
	procGetLongPathNameW         = kernel32.NewProc("GetLongPathNameW")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = kernel32.NewProc("Process32FirstW")
	procProcess32Next            = kernel32.NewProc("Process32NextW")

	// advapi32 ETW consumer APIs were previously bound to kernel32, which
	// does not export them — on real Windows that caused TestEtwSessionLifecycle
	// to panic with "procedure could not be found" instead of failing closed.
	procOpenTraceW   = advapi32.NewProc("OpenTraceW")
	procProcessTrace = advapi32.NewProc("ProcessTrace")
	procCloseTrace   = advapi32.NewProc("CloseTrace")

	// WFP procs. Bound lazily after initWFP() picks the right DLL.
	// Versioned names (FwpmXxx0) are used because the C-header #define macros
	// (FwpmEngineOpen without the "0" suffix) are not present in the export
	// table — using unversioned names causes "procedure could not be found"
	// at Call time, which would have panicked under the old single-DLL binding.
	procFwpmEngineOpen        *windows.LazyProc
	procFwpmEngineClose       *windows.LazyProc
	procFwpmTransactionBegin  *windows.LazyProc
	procFwpmTransactionCommit *windows.LazyProc
	procFwpmTransactionAbort  *windows.LazyProc
	procFwpmSublayerAdd       *windows.LazyProc
	procFwpmFilterAdd         *windows.LazyProc
	procFwpmFreeMemory        *windows.LazyProc
	procFwpmFilterDeleteById  *windows.LazyProc
)

// initWFP picks the Windows DLL that actually exports the WFP user-mode API on
// this host and binds the wfp* proc pointers against it. Both fwpuclnt.dll
// (the canonical home of the API since Vista) and iphlapi.dll (which also
// exports the same symbols on stripped server SKUs and the GitHub Actions
// `windows-latest` runner image) are attempted.
//
// Important: Load() succeeding is not enough. On some images fwpuclnt.dll is
// present but does not export the versioned Fwpm*0 entry points; selecting it
// then makes every Run fail with "procedure could not be found". We require
// Find() to succeed for every required symbol before committing to a DLL.
func initWFP() {
	required := []string{
		"FwpmEngineOpen0",
		"FwpmEngineClose0",
		"FwpmTransactionBegin0",
		"FwpmTransactionCommit0",
		"FwpmTransactionAbort0",
		"FwpmSublayerAdd0",
		"FwpmFilterAdd0",
		"FwpmFreeMemory0",
		"FwpmFilterDeleteById0",
	}
	candidates := []string{"fwpuclnt.dll", "iphlapi.dll"}
	for _, name := range candidates {
		d := windows.NewLazySystemDLL(name)
		if err := d.Load(); err != nil {
			continue
		}
		procs := make([]*windows.LazyProc, len(required))
		ok := true
		for i, sym := range required {
			p := d.NewProc(sym)
			if err := p.Find(); err != nil {
				ok = false
				break
			}
			procs[i] = p
		}
		if !ok {
			continue
		}
		wfpDLL = d
		procFwpmEngineOpen = procs[0]
		procFwpmEngineClose = procs[1]
		procFwpmTransactionBegin = procs[2]
		procFwpmTransactionCommit = procs[3]
		procFwpmTransactionAbort = procs[4]
		procFwpmSublayerAdd = procs[5]
		procFwpmFilterAdd = procs[6]
		procFwpmFreeMemory = procs[7]
		procFwpmFilterDeleteById = procs[8]
		return
	}
}

// wfpDLLName reports which DLL currently backs the WFP procs. Returns
// "(none)" when initWFP() could not find a host for the API. Used by
// wfpSupported() for actionable error messages.
func wfpDLLName() string {
	if wfpDLL == nil {
		return "(none)"
	}
	return wfpDLL.Name
}

func init() {
	initWFP()
}

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

// _GetLongPathNameW expands 8.3 short path components (e.g. RUNNER~1) to the
// long form via kernel32.GetLongPathNameW. Returns the required buffer length
// including the terminating NUL on success; 0 on failure.
func _GetLongPathNameW(short, long *uint16, bufLen uint32) (uint32, error) {
	r, _, e := procGetLongPathNameW.Call(
		uintptr(unsafe.Pointer(short)),
		uintptr(unsafe.Pointer(long)),
		uintptr(bufLen),
	)
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

// errWFPDLLMissing is returned by wfpSupported() when neither fwpuclnt.dll
// nor iphlapi.dll could be loaded. Surfaced verbatim so the user can fix the
// host or report the issue; the run fails closed with this message rather
// than panicking inside LazyProc.Call (the original REMAINING_WORK P0 bug).
var errWFPDLLMissing = errors.New(
	"neither fwpuclnt.dll nor iphlapi.dll could be loaded — the Windows Filtering Platform user-mode API is unavailable on this host",
)
