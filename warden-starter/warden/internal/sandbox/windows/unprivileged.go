//go:build windows

package windows

import (
	"errors"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/warden-sandbox/warden/internal/policy"
)

// Unprivileged mode lets a non-administrator run the Windows backend without
// the two layers that need elevation: WFP egress filters and the kernel ETW
// audit session. It is automatic for offline policies (and can be explicitly
// selected with WARDEN_WINDOWS_UNPRIVILEGED=1) and never widens the boundary:
//   - Filesystem isolation is unchanged: the AppContainer token plus DACL
//     grants on the policy paths, which a normal user can set on paths they
//     own.
//   - Network: the token carries no internetClient/privateNetwork capability,
//     so Windows' own AppContainer network isolation blocks every outbound
//     connection, loopback included. Without WFP there is no way to permit
//     only the proxy, so a policy with a network allowlist is refused.
//   - Audit: file-activity ETW is unavailable; the run records that in the
//     audit log instead of silently dropping it.
//   - Job Object limits and process-tree cleanup are unchanged.
const unprivilegedEnv = "WARDEN_WINDOWS_UNPRIVILEGED"

var errUnprivilegedNetwork = errors.New("network.allow needs WFP egress filtering, which requires an administrator terminal; remove network.allow to run with all network blocked")

// unprivilegedOptIn reports whether the user explicitly chose unprivileged mode.
func unprivilegedOptIn() bool {
	return os.Getenv(unprivilegedEnv) == "1"
}

// checkUnprivilegedPolicy refuses policies unprivileged mode cannot enforce.
func checkUnprivilegedPolicy(p policy.Policy) error {
	if len(p.Network.Allow) > 0 {
		return failClose("unprivileged Windows sandbox", errUnprivilegedNetwork)
	}
	return nil
}

// procThreadAttributeSecurityCapabilities is
// PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES (processthreadsapi.h).
const procThreadAttributeSecurityCapabilities = 0x00020009

// securityCapabilities mirrors SECURITY_CAPABILITIES (winnt.h).
type securityCapabilities struct {
	AppContainerSid *windows.SID
	Capabilities    *sidAndAttributes
	CapabilityCount uint32
	Reserved        uint32
}

// createProcessAppContainer launches the plan's command line inside the
// plan's AppContainer without a pre-built token: CreateProcess with
// PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES lets the kernel build the
// LowBox token itself (the profile must be registered first, see
// registerAppContainerProfile), which needs no SeImpersonate/SeAssignPrimaryToken
// privilege (unlike CreateProcessWithTokenW). Only the three stdio handles
// are inherited, through an explicit handle list.
func createProcessAppContainer(plan *Plan) (*processInformation, error) {
	pkg, err := windows.StringToSid(plan.PackageSID)
	if err != nil {
		return nil, failClose("parse package SID "+plan.PackageSID, err)
	}
	caps := make([]sidAndAttributes, 0, len(plan.CapabilitySIDs))
	for _, s := range plan.CapabilitySIDs {
		sid, err := windows.StringToSid(s)
		if err != nil {
			return nil, failClose("parse capability SID "+s, err)
		}
		caps = append(caps, sidAndAttributes{Sid: sid, Attributes: capabilityAttribute})
	}
	sc := securityCapabilities{AppContainerSid: pkg, CapabilityCount: uint32(len(caps))}
	if len(caps) > 0 {
		sc.Capabilities = &caps[0]
	}

	var handles []windows.Handle
	for _, kind := range []uint32{stdInputHandle, stdOutputHandle, stdErrorHandle} {
		h := stdHandle(kind)
		if h == 0 || h == windows.InvalidHandle {
			continue
		}
		if err := windows.SetHandleInformation(h, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); err != nil {
			continue // console pseudo-handles cannot be marked; the console is shared anyway
		}
		dup := false
		for _, x := range handles {
			dup = dup || x == h
		}
		if !dup {
			handles = append(handles, h)
		}
	}

	attrs, err := windows.NewProcThreadAttributeList(2)
	if err != nil {
		return nil, failClose("process attribute list", err)
	}
	defer attrs.Delete()
	if err := attrs.Update(procThreadAttributeSecurityCapabilities, unsafe.Pointer(&sc), unsafe.Sizeof(sc)); err != nil {
		return nil, failClose("AppContainer security capabilities", err)
	}
	inherit := len(handles) > 0
	if inherit {
		if err := attrs.Update(windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST, unsafe.Pointer(&handles[0]), uintptr(len(handles))*unsafe.Sizeof(handles[0])); err != nil {
			return nil, failClose("inherited handle list", err)
		}
	}

	si := windows.StartupInfoEx{ProcThreadAttributeList: attrs.List()}
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Flags = windows.STARTF_USESTDHANDLES
	si.StdInput = stdHandle(stdInputHandle)
	si.StdOutput = stdHandle(stdOutputHandle)
	si.StdErr = stdHandle(stdErrorHandle)

	var pi windows.ProcessInformation
	// Suspended: Run assigns the Job Object and starts auditing, then resumes.
	flags := uint32(createUnicodeEnvironment | normalPriorityClass | windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_SUSPENDED)
	if err := windows.CreateProcess(nil, utf16Ptr(plan.CommandLine), nil, nil, inherit, flags,
		envBlockPtr(withContainerEnv(plan.EnvBlock)), workDirPtr(plan.WorkDir), &si.StartupInfo, &pi); err != nil {
		return nil, failClose("CreateProcess (AppContainer)", err)
	}
	return &processInformation{hProcess: pi.Process, hThread: pi.Thread, dwProcessId: pi.ProcessId, dwThreadId: pi.ThreadId}, nil
}

var (
	userenv                       = windows.NewLazySystemDLL("userenv.dll")
	procCreateAppContainerProfile = userenv.NewProc("CreateAppContainerProfile")
	procDeleteAppContainerProfile = userenv.NewProc("DeleteAppContainerProfile")
)

// hresultAlreadyExists is HRESULT_FROM_WIN32(ERROR_ALREADY_EXISTS).
const hresultAlreadyExists = 0x800700B7

// registerAppContainerProfile creates the per-run AppContainer profile that
// CreateProcess needs for the plan's package SID (a normal user may do this).
// The returned func deletes it again.
func registerAppContainerProfile(plan *Plan) (func(), error) {
	name := utf16Ptr(plan.AppContainerName)
	var sid *windows.SID
	r, _, _ := procCreateAppContainerProfile.Call(uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(name)), 0, 0, uintptr(unsafe.Pointer(&sid)))
	if r != 0 && uint32(r) != hresultAlreadyExists {
		return nil, failClose("CreateAppContainerProfile", windows.Errno(r&0xffff))
	}
	if sid != nil {
		got := sid.String()
		_ = windows.FreeSid(sid)
		if got != plan.PackageSID {
			_, _, _ = procDeleteAppContainerProfile.Call(uintptr(unsafe.Pointer(name)))
			return nil, failClose("CreateAppContainerProfile", errors.New("profile SID "+got+" does not match plan SID "+plan.PackageSID))
		}
	}
	return func() { _, _, _ = procDeleteAppContainerProfile.Call(uintptr(unsafe.Pointer(name))) }, nil
}


// containerEnvVars are the per-user folder variables CreateProcess rewrites to
// the AppContainer's private folders; it fails with ERROR_ENVVAR_NOT_FOUND
// when the (deny-by-default) environment block lacks them. Their values are
// replaced by Windows, so the host paths are never exposed to the target.
var containerEnvVars = []string{"LOCALAPPDATA"}

// withContainerEnv adds any missing containerEnvVars from the host to block.
func withContainerEnv(block string) string {
	have := map[string]bool{}
	for _, kv := range strings.Split(block, "\x00") {
		if name, _, ok := strings.Cut(kv, "="); ok {
			have[strings.ToUpper(name)] = true
		}
	}
	for _, name := range containerEnvVars {
		if v, ok := os.LookupEnv(name); ok && !have[strings.ToUpper(name)] {
			block = name + "=" + v + "\x00" + block
		}
	}
	return block
}

// workDirPtr returns dir as a UTF-16 pointer, or nil (inherit) when empty.
func workDirPtr(dir string) *uint16 {
	if dir == "" {
		return nil
	}
	return utf16Ptr(longPath(dir))
}
