//go:build windows

package windows

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/policy"
)

// This file implements the Job Object that constrains the sandboxed process
// tree: a wall-clock timeout (Go timer), a hard memory cap (Job Object limit so
// allocations fail closed), and kill-on-close so the whole tree dies however
// the run exits. Every limit must be applied before the target spawns; a partial
// Job Object is a fail-closed gate.

// Job Object information classes (winnt.h).
const jobObjectExtendedLimitInfoClass = 9

// JOB_OBJECT_LIMIT_* flags (winnt.h).
const (
	jobObjectLimitKillOnJobClose = 0x00002000
	jobObjectLimitJobMemory      = 0x00000200
)

type ioCounters struct {
	readOpCount        uint64
	writeOpCount       uint64
	otherOpCount       uint64
	readTransferCount  uint64
	writeTransferCount uint64
	otherTransferCount uint64
}

// jobObjectBasicLimitInformation mirrors JOBOBJECT_BASIC_LIMIT_INFORMATION
// (winnt.h). Fields are not manually padded; the Go compiler inserts the same
// alignment as MSVC x64 for the uint64/uintptr members.
type jobObjectBasicLimitInformation struct {
	perProcessUserTimeLimit int64   // LARGE_INTEGER
	perJobUserTimeLimit     int64   // LARGE_INTEGER
	limitFlags              uint32  // DWORD
	minimumWorkingSetSize   uint64  // SIZE_T
	maximumWorkingSetSize   uint64  // SIZE_T
	activeProcessLimit      uint32  // DWORD
	affinity                uintptr // ULONG_PTR
	priorityClass           uint32  // DWORD
	schedulingClass         uint32  // DWORD
}

// jobObjectExtendedLimitInformation mirrors
// JOBOBJECT_EXTENDED_LIMIT_INFORMATION (winnt.h).
type jobObjectExtendedLimitInformation struct {
	basicLimitInformation jobObjectBasicLimitInformation
	ioInfo                ioCounters // 6 * LONGLONG
	processMemoryLimit    uint64     // SIZE_T
	jobMemoryLimit        uint64     // SIZE_T
	peakProcessMemoryUsed uint64     // SIZE_T
	peakJobMemoryUsed     uint64     // SIZE_T
}

// LimitExceededError reports an enforced policy limit, matching the Linux and
// macOS backends so the CLI treats a breach uniformly.
type LimitExceededError struct {
	Kind  string
	Limit string
}

func (e *LimitExceededError) Error() string {
	return fmt.Sprintf("resource limit exceeded: %s (%s)", e.Kind, e.Limit)
}

// jobObject wraps a Job Object handle with kill-on-close semantics.
type jobObject struct {
	handle windows.Handle
}

// configureJobObject creates a Job Object and applies the policy limits. The
// returned object kills the whole process tree when its handle is closed and
// refuses memory growth above memory_mb.
func configureJobObject(limits policy.Limits, logger *audit.Logger) (*jobObject, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, failClose("create Job Object", err)
	}
	j := &jobObject{handle: h}

	info := jobObjectExtendedLimitInformation{}
	info.basicLimitInformation.limitFlags = jobObjectLimitKillOnJobClose
	if limits.MemoryMB > 0 {
		info.basicLimitInformation.limitFlags |= jobObjectLimitJobMemory
		info.jobMemoryLimit = uint64(limits.MemoryMB) * 1024 * 1024
	}
	if _, err := windows.SetInformationJobObject(
		j.handle,
		jobObjectExtendedLimitInfoClass,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = j.Close()
		return nil, failClose("configure Job Object limits", err)
	}
	return j, nil
}

// Close terminates the process tree (KILL_ON_JOB_CLOSE) and frees the handle.
func (j *jobObject) Close() error {
	if j == nil || j.handle == 0 {
		return nil
	}
	err := windows.CloseHandle(j.handle)
	j.handle = 0
	return err
}

// Terminate kills the whole sandboxed process tree.
func (j *jobObject) terminate() {
	_ = windows.TerminateJobObject(j.handle, 1)
}

// peakMemory returns the job's peak committed memory (PeakJobMemoryUsed).
func (j *jobObject) peakMemory() (uint64, error) {
	var info jobObjectExtendedLimitInformation
	if err := windows.QueryInformationJobObject(
		j.handle,
		jobObjectExtendedLimitInfoClass,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
		nil,
	); err != nil {
		return 0, err
	}
	return info.peakJobMemoryUsed, nil
}

// wait blocks until the target process exits, enforcing the wall-clock timeout
// and hard memory cap across the whole process tree. On a breach it terminates
// the Job Object (killing every descendant) and returns a LimitExceededError;
// otherwise it returns the target's exit code. A breach is also recorded in the
// audit log so enforcement decisions are never silent.
func (j *jobObject) wait(process windows.Handle, limits policy.Limits, logger *audit.Logger) (int, *LimitExceededError) {
	done := make(chan int, 1)
	go func() {
		if _, err := windows.WaitForSingleObject(process, windows.INFINITE); err != nil {
			done <- -1
			return
		}
		var code uint32
		if err := windows.GetExitCodeProcess(process, &code); err != nil {
			done <- -1
			return
		}
		done <- int(code)
	}()

	var timeout <-chan time.Time
	if limits.TimeoutS > 0 {
		t := time.NewTimer(time.Duration(limits.TimeoutS) * time.Second)
		defer t.Stop()
		timeout = t.C
	}
	var ticks <-chan time.Time
	if limits.MemoryMB > 0 {
		tk := time.NewTicker(25 * time.Millisecond)
		defer tk.Stop()
		ticks = tk.C
	}
	maxMem := uint64(limits.MemoryMB) * 1024 * 1024

	for {
		select {
		case code := <-done:
			return code, nil
		case <-timeout:
			j.terminate()
			if logger != nil {
				_ = logger.Log(audit.Event{Type: "limit", Action: "terminate", Resource: fmt.Sprintf("%ds", limits.TimeoutS), Allowed: false, Reason: "wall-clock timeout"})
			}
			<-done
			return 0, &LimitExceededError{Kind: "wall-clock timeout", Limit: fmt.Sprintf("%ds", limits.TimeoutS)}
		case <-ticks:
			used, err := j.peakMemory()
			// The hard Job Object limit already blocks allocations; this
			// poller converts a stalled process into a clean tree-termination
			// and an audit record. Wait for the exit it triggers.
			if err == nil && used > maxMem {
				j.terminate()
				if logger != nil {
					_ = logger.Log(audit.Event{Type: "limit", Action: "terminate", Resource: fmt.Sprintf("%dMB", limits.MemoryMB), Allowed: false, Reason: "memory"})
				}
				<-done
				return 0, &LimitExceededError{Kind: "memory", Limit: fmt.Sprintf("%dMB", limits.MemoryMB)}
			}
		}
	}
}
