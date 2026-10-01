package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/warden-sandbox/warden/internal/policy"
)

// canonicalOutput resolves existing ancestors even before an output exists.
// Private state must not land in a granted tree through a directory alias.
func canonicalOutput(path string) (string, error) {
	abs, e := filepath.Abs(path)
	if e != nil {
		return "", e
	}
	probe := abs
	var tail []string
	for {
		_, e = os.Lstat(probe)
		if e == nil {
			break
		}
		if !os.IsNotExist(e) || filepath.Dir(probe) == probe {
			return "", e
		}
		tail = append([]string{filepath.Base(probe)}, tail...)
		probe = filepath.Dir(probe)
	}
	real, e := filepath.EvalSymlinks(probe)
	if e != nil {
		return "", e
	}
	return filepath.Join(append([]string{real}, tail...)...), nil
}

// This is a conservative private-state guard, not a grant added to a backend.
// Its runtime directories mirror the existing backends; Linux/Docker /tmp is
// fresh scratch, while Seatbelt exposes the host temporary directories.
func outsideUpstream(path string, p policy.Policy, command []string, backend string) error {
	real, e := canonicalOutput(path)
	if e != nil {
		return e
	}
	grants := append(append([]string{}, p.Filesystem.Read...), p.Filesystem.Write...)
	if len(command) > 0 {
		cmd, e := policy.ResolveExecutable(command)
		if e != nil {
			return e
		}
		grants = append(grants, filepath.Dir(cmd[0]))
	}
	switch backend {
	case "linux":
		grants = append(grants, "/usr", "/lib", "/lib64")
	case "docker":
		grants = append(grants, "/usr", "/lib", "/lib64", "/bin")
	case "seatbelt":
		grants = append(grants, "/usr", "/bin", "/sbin", "/System", "/Library/Frameworks", "/Library/Apple", "/opt/homebrew", "/private/var/db/dyld", "/private/var/db/timezone", "/Library/Preferences/Logging", "/etc", "/private/etc", "/dev", "/private/tmp", "/tmp", os.TempDir())
	case "windows":
		grants = append(grants, `C:\Windows\System32`, `C:\Windows\SysWOW64`, `C:\Windows\WinSxS`, `C:\Windows\Fonts`, `C:\Windows\Globalization`)
	}
	for _, grant := range grants {
		resolved, e := canonicalOutput(grant)
		if e != nil {
			return fmt.Errorf("cannot resolve private-state boundary: %w", e)
		}
		if policy.Within(real, resolved) {
			return fmt.Errorf("private state must be outside upstream grants, executable directory and runtime paths")
		}
	}
	return nil
}
