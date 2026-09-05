//go:build windows

package windows

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// processEntry32W mirrors PROCESSENTRY32W (tlhelp32.h).
type processEntry32W struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

const (
	th32CS_SNAPPROCESS = 0x00000002
)

// snapshotProcesses returns a full system process table via the Toolhelp32
// API. Callers must CloseHandle the snapshot.
func snapshotProcesses() (windows.Handle, error) {
	return _CreateToolhelp32Snapshot(th32CS_SNAPPROCESS, 0)
}

// processTreePIDs returns the set of PIDs in the process tree rooted at root
// (root and all descendants), computed from a parent-PID relation snapshot.
// The ETW auditor and the memory monitor use this to scope events to the
// sandbox and to make termination reach every child.
func processTreePIDs(root uint32) (map[uint32]bool, error) {
	snap, err := snapshotProcesses()
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snap)

	parent := make(map[uint32]uint32)
	var entry processEntry32W
	entry.Size = uint32(unsafe.Sizeof(entry))
	first := true
	for {
		var err error
		if first {
			err = _Process32First(snap, &entry)
			first = false
		} else {
			err = _Process32Next(snap, &entry)
		}
		if err != nil {
			// ERROR_NO_MORE_FILES ends the walk; anything else is fatal.
			if errno, ok := err.(windows.Errno); ok && int(errno) == 18 { // ERROR_NO_MORE_FILES
				break
			}
			return nil, err
		}
		parent[entry.ProcessID] = entry.ParentProcessID
	}

	seen := map[uint32]bool{root: true}
	changed := true
	for changed {
		changed = false
		for pid, ppid := range parent {
			if seen[pid] || !seen[ppid] {
				continue
			}
			seen[pid] = true
			changed = true
		}
	}
	return seen, nil
}
