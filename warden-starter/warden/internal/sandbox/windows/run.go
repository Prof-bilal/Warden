//go:build windows

package windows

import (
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/proxy"
)

// Run sandboxes cmd under an AppContainer token with the policy enforced by
// the filesystem capabilities, WFP egress filters, ETW audit, and a Job Object
// limits. Every enforcement layer is applied before the target starts and a
// failure in any of them refuses to run (fail closed). The command's stdio is
// passed through transparently so an MCP client sees the sandboxed server
// exactly as it would the unsandboxed binary.
func Run(cmd []string, p policy.Policy) (int, error) {
	logFile, _, err := audit.OpenDefault()
	if err != nil {
		return 0, err
	}
	defer logFile.Close()
	logger := audit.New(logFile)

	// Host-side egress proxy on a loopback TCP address; its exact endpoint is
	// the only destination WFP permits.
	eg, err := proxy.StartTCP(p.Network.Allow, logger)
	if err != nil {
		return 0, err
	}
	defer eg.Close()

	plan, err := BuildPlan(cmd, p, NewSessionID(), eg.Addr())
	if err != nil {
		return 0, err
	}

	// 1. AppContainer token.
	token, err := makeLowBoxToken(plan)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(token)

	// 2. Filesystem capabilities (DACL grants on the policy paths).
	if err := grantFilesystemAccess(plan); err != nil {
		return 0, err
	}

	// 3. WFP egress filters: permit loopback-to-proxy, block the rest.
	image, err := imageNTPath(plan.Command[0])
	if err != nil {
		return 0, failClose("resolve image path for WFP", err)
	}
	wfp, err := installWFPEgress(image, plan.AllowPorts[0], plan.AppContainerName)
	if err != nil {
		return 0, err
	}
	defer wfp.close()

	// 4. Job Object limits (wall-clock + memory + kill-on-close).
	job, err := configureJobObject(p.Limits, logger)
	if err != nil {
		return 0, err
	}
	defer job.Close()

	// 5. Start ETW auditing before the target begins.
	trace, err := startTrace("warden-"+plan.AppContainerName, nil)
	if err != nil {
		return 0, err
	}

	// 6. Spawn the target under the LowBox token.
	pi, err := createProcess(plan, token)
	if err != nil {
		_ = trace.closeSession()
		return 0, err
	}
	defer windows.CloseHandle(pi.hThread)
	defer windows.CloseHandle(pi.hProcess)

	// Pin the whole process tree to the Job Object so limits and kill-on-close
	// reach every descendant, and scope ETW records to the tree.
	if err := windows.AssignProcessToJobObject(job.handle, pi.hProcess); err != nil {
		_ = trace.closeSession()
		return 0, failClose("assign process tree to Job Object", err)
	}
	tree, _ := processTreePIDs(pi.dwProcessId)
	trace.tree = tree
	trace.logger = logger
	go trace.run()

	code, limitErr := job.wait(pi.hProcess, p.Limits, logger)
	_ = trace.closeSession()
	if limitErr != nil {
		return 0, limitErr
	}
	return code, nil
}

// createProcess builds the startup/environment/command-line and launches the
// plan's command line under the LowBox token with inherited stdio.
func createProcess(plan *Plan, token windows.Handle) (*processInformation, error) {
	si := startupInfo{}
	si.cb = uint32(unsafe.Sizeof(si))
	si.hStdInput = stdHandle(stdInputHandle)
	si.hStdOutput = stdHandle(stdOutputHandle)
	si.hStdError = stdHandle(stdErrorHandle)
	si.dwFlags = startfUseStdHandles

	env := envBlockPtr(plan.EnvBlock)
	cmdLine := utf16Ptr(plan.CommandLine)

	var pi processInformation
	flags := uint32(createUnicodeEnvironment | normalPriorityClass)
	err := _CreateProcessWithTokenW(token, flags, cmdLine, env, nil, &si, &pi)
	if err != nil {
		return nil, failClose("CreateProcessWithTokenW", err)
	}
	return &pi, nil
}

func stdHandle(kind uint32) windows.Handle {
	h, err := windows.GetStdHandle(kind)
	if err != nil {
		return 0
	}
	return h
}

// Standard-handle identifiers (winbase.h). These are negative ints found via
// GetStdHandle; expressed as their two's-complement bit patterns.
const (
	stdInputHandle  = ^uint32(9)  // (DWORD)-10
	stdOutputHandle = ^uint32(10) // (DWORD)-11
	stdErrorHandle  = ^uint32(11) // (DWORD)-12
)

// startupInfo mirrors STARTUPINFOW (processthreadsapi.h).
type startupInfo struct {
	cb              uint32
	lpReserved      *uint16
	lpDesktop       *uint16
	lpTitle         *uint16
	dwX             uint32
	dwY             uint32
	dwXSize         uint32
	dwYSize         uint32
	dwXCountChars   uint32
	dwYCountChars   uint32
	dwFillAttribute uint32
	dwFlags         uint32
	wShowWindow     uint16
	cbReserved2     uint16
	lpReserved2     *byte
	hStdInput       windows.Handle
	hStdOutput      windows.Handle
	hStdError       windows.Handle
}

// processInformation mirrors PROCESS_INFORMATION (processthreadsapi.h).
type processInformation struct {
	hProcess    windows.Handle
	hThread     windows.Handle
	dwProcessId uint32
	dwThreadId  uint32
}

// CreateProcess creation flags.
const (
	normalPriorityClass      = 0x00000020
	createUnicodeEnvironment = 0x00000400
	startfUseStdHandles      = 0x00000100
)

// envBlockPtr turns the plan's NUL-separated environment block into a stable
// UTF-16 buffer ending in a double NUL.
func envBlockPtr(block string) *uint16 {
	u := utf16.Encode([]rune(block))
	u = append(u, 0, 0)
	return &u[0]
}
