//go:build windows

package windows

import (
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/warden-sandbox/warden/internal/approve"
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

	return runWithApproval(cmd, p, logger, nil)
}

// RunWithApproval sandboxes cmd like Run with interactive approval mode for
// network requests: blocked hosts prompt on the terminal and approved ones
// apply live via the egress proxy. ETW audit events are post-hoc, so there
// is no live filesystem signalfilesystem stays hard-deny (use
// `warden trace` + `warden init` to widen it). A nil or disabled cfg behaves
// like Run.
func RunWithApproval(cmd []string, p policy.Policy, cfg *approve.Config) (int, error) {
	if cfg == nil || !cfg.Enabled {
		return Run(cmd, p)
	}
	if err := cfg.Validate(); err != nil {
		return 0, err
	}
	logFile, _, err := audit.OpenDefault()
	if err != nil {
		return 0, err
	}
	defer logFile.Close()
	return runWithApproval(cmd, p, audit.New(logFile), cfg)
}

func runWithApproval(cmd []string, p policy.Policy, logger *audit.Logger, approval *approve.Config) (int, error) {
	// Host-side egress proxy on a loopback TCP address; its exact endpoint is
	// the only destination WFP permits.
	// Without WFP (no elevation) only the explicit unprivileged mode may run,
	// and only for policies it can fully enforce.
	unprivileged := false
	if werr := wfpSupported(); werr != nil {
		if !unprivilegedOptIn() {
			return 0, werr
		}
		if err := checkUnprivilegedPolicy(p); err != nil {
			return 0, err
		}
		unprivileged = true
	}

	eg, err := proxy.StartTCP(p.Network.Allow, logger)
	if err != nil {
		return 0, err
	}
	defer eg.Close()

	if approval != nil && approval.Enabled {
		eg.SetApprover(approve.NewPrompter(approval.PolicyPath, approval.Timeout, logger).NetworkApprover())
	}

	plan, err := BuildPlan(cmd, p, NewSessionID(), eg.Addr())
	if err != nil {
		return 0, err
	}

	// 1. AppContainer identity: the kernel builds the LowBox token from the
	// plan's package SID and capabilities at CreateProcess (step 6).

	// The AppContainer profile must stay registered for the whole lifetime of
	// the target.
	cleanupProfile, err := registerAppContainerProfile(plan)
	if err != nil {
		return 0, err
	}
	defer cleanupProfile()

	// 2. Filesystem capabilities (DACL grants on the policy paths).
	restoreACLs, err := grantFilesystemAccess(plan)
	defer restoreACLs()
	if err != nil {
		return 0, err
	}
	// Read-attributes on the grants' ancestors so runtimes can resolve real
	// paths (see ancestors.go).
	defer grantAncestorAttributes(plan)()

	// 3. WFP egress filters: permit loopback-to-proxy, block the rest. In
	// unprivileged mode the token has no network capability, so AppContainer
	// isolation itself blocks all egress (see unprivileged.go).
	if !unprivileged {
		wfp, err := installWFPEgress(plan.PackageSID, plan.AllowPorts[0], plan.AppContainerName)
		if err != nil {
			return 0, err
		}
		defer wfp.close()

		// AppContainers cannot reach loopback at all unless exempted; the
		// exemption is scoped to this run's package SID and WFP still permits
		// only the proxy port.
		restoreLoopback, err := exemptLoopback(plan.PackageSID)
		if err != nil {
			return 0, err
		}
		defer restoreLoopback()
	}

	// 4. Job Object limits (wall-clock + memory + kill-on-close).
	job, err := configureJobObject(p.Limits, logger)
	if err != nil {
		return 0, err
	}
	defer job.Close()

	// 5. Start ETW auditing before the target begins. Fail closed: an audit
	// session that cannot start refuses the whole run (the security gate), so
	// the trace's error is returned as-is.
	var trace *traceSession
	if unprivileged {
		_ = logger.Log(audit.Event{
			Timestamp: time.Now(),
			Type:      "sandbox",
			Action:    "unprivileged",
			Resource:  plan.AppContainerName,
			Allowed:   true,
			Reason:    "Windows unprivileged mode: ETW file audit and WFP disabled; all network blocked by AppContainer",
		})
	} else {
		trace, err = startTrace("warden-"+plan.AppContainerName, logger)
		if err != nil {
			return 0, err
		}
	}

	// 6. Spawn the target under the LowBox token.
	// Both modes launch through CreateProcess + SECURITY_CAPABILITIES with a
	// registered profile: a raw NtCreateLowBoxToken token has no AppContainer
	// named-object directory, so the target dies in loader init (0xC0000142).
	pi, err := createProcessAppContainer(plan)
	if err != nil {
		trace.closeIfOpen()
		return 0, err
	}
	defer windows.CloseHandle(pi.hThread)
	defer windows.CloseHandle(pi.hProcess)

	// The target was created suspended: pin it to the Job Object and start
	// the ETW consumer before its first instruction, so no descendant can
	// escape the limits and no early file activity is missed.
	if err := windows.AssignProcessToJobObject(job.handle, pi.hProcess); err != nil {
		_ = windows.TerminateProcess(pi.hProcess, 1)
		trace.closeIfOpen()
		return 0, failClose("assign process tree to Job Object", err)
	}
	if trace != nil {
		tree, _ := processTreePIDs(pi.dwProcessId)
		trace.tree = tree
		go trace.run()
	}
	if _, err := windows.ResumeThread(pi.hThread); err != nil {
		_ = windows.TerminateProcess(pi.hProcess, 1)
		trace.closeIfOpen()
		return 0, failClose("resume sandboxed process", err)
	}

	code, limitErr := job.wait(pi.hProcess, p.Limits, logger)
	trace.closeIfOpen()
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

// closeIfOpen stops the ETW session when one was started (nil in
// unprivileged mode).
func (t *traceSession) closeIfOpen() {
	if t != nil {
		_ = t.closeSession()
	}
}
