// Package sandbox selects and invokes the secure backend for a run.
// Selection prefers the OS-native primitive; Docker is only used when the
// native backend is missing (or when --backend docker is passed explicitly).
// Warden never falls back to an unsandboxed execution.
package sandbox

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/sandbox/docker"
	windowsbackend "github.com/warden-sandbox/warden/internal/sandbox/windows"
)

// Canonical backend names accepted by Resolve / the CLI --backend flag.
const (
	BackendAuto     = "auto"
	BackendLinux    = "linux"
	BackendSeatbelt = "seatbelt"
	BackendWindows  = "windows"
	BackendDocker   = "docker"
)

// Resolve picks the backend to use. An empty name or "auto" means detect the
// best available backend for this host. Explicit names are validated and fail
// closed when the requested primitive is not available.
func Resolve(name string) (string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || name == BackendAuto {
		return detect()
	}
	switch name {
	case BackendLinux, "bwrap":
		if !linuxAvailable() {
			return "", fmt.Errorf("backend %q requested but bwrap is not available on this host", BackendLinux)
		}
		if runtime.GOOS != "linux" {
			return "", fmt.Errorf("backend %q is only available on Linux (this host is %s)", BackendLinux, runtime.GOOS)
		}
		return BackendLinux, nil
	case BackendSeatbelt, "macos", "darwin", "sandbox-exec":
		if !seatbeltAvailable() {
			return "", fmt.Errorf("backend %q requested but sandbox-exec is not available on this host", BackendSeatbelt)
		}
		if runtime.GOOS != "darwin" {
			return "", fmt.Errorf("backend %q is only available on macOS (this host is %s)", BackendSeatbelt, runtime.GOOS)
		}
		return BackendSeatbelt, nil
	case BackendWindows, "appcontainer", "lowbox":
		if !windowsAvailable() {
			return "", fmt.Errorf("backend %q requested but AppContainer sandbox support is not available on this host", BackendWindows)
		}
		if runtime.GOOS != "windows" {
			return "", fmt.Errorf("backend %q is only available on Windows (this host is %s)", BackendWindows, runtime.GOOS)
		}
		return BackendWindows, nil
	case BackendDocker:
		if !dockerAvailable() {
			return "", fmt.Errorf("backend %q requested but docker is not available or not usable on this host", BackendDocker)
		}
		return BackendDocker, nil
	default:
		return "", fmt.Errorf("unknown backend %q (want auto, linux, seatbelt, windows, or docker)", name)
	}
}

// detect returns the preferred available backend for this OS.
// Preference: native (bwrap / sandbox-exec) then Docker. Never unsandboxed.
func detect() (string, error) {
	switch runtime.GOOS {
	case "linux":
		if linuxAvailable() {
			return BackendLinux, nil
		}
		if dockerAvailable() {
			return BackendDocker, nil
		}
		return "", fmt.Errorf("no sandbox backend available on linux: need bwrap (preferred) or docker; refusing to run unsandboxed")
	case "darwin":
		if seatbeltAvailable() {
			return BackendSeatbelt, nil
		}
		if dockerAvailable() {
			return BackendDocker, nil
		}
		return "", fmt.Errorf("no sandbox backend available on macOS: need sandbox-exec (preferred) or docker; refusing to run unsandboxed")
	case "windows":
		// AppContainer is the OS-native primitive; Docker is only a fallback.
		if windowsAvailable() {
			return BackendWindows, nil
		}
		if dockerAvailable() {
			return BackendDocker, nil
		}
		return "", fmt.Errorf("no sandbox backend available on Windows: need AppContainer support (preferred) or docker; refusing to run unsandboxed")
	default:
		if dockerAvailable() {
			return BackendDocker, nil
		}
		return "", fmt.Errorf("no sandbox backend available on %s: need docker; refusing to run unsandboxed", runtime.GOOS)
	}
}

func linuxAvailable() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	_, err := exec.LookPath("bwrap")
	return err == nil
}

func seatbeltAvailable() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	_, err := exec.LookPath("sandbox-exec")
	return err == nil
}

func windowsAvailable() bool {
	return windowsbackend.Supported()
}

func dockerAvailable() bool {
	return docker.Available()
}

// Run executes cmd under the resolved sandbox backend.
// backend may be empty/"auto" or an explicit backend name.
func Run(cmd []string, p policy.Policy, backend string) (int, error) {
	name, err := Resolve(backend)
	if err != nil {
		return 0, err
	}
	switch name {
	case BackendLinux:
		return runLinux(cmd, p)
	case BackendSeatbelt:
		return runSeatbelt(cmd, p)
	case BackendWindows:
		return runWindows(cmd, p)
	case BackendDocker:
		return docker.Run(cmd, p)
	default:
		return 0, fmt.Errorf("internal error: unresolved backend %q", name)
	}
}
