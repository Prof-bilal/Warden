package sandbox

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/sandbox/sandboxerr"
)

func TestResolveAutoPrefersNative(t *testing.T) {
	name, err := Resolve("auto")
	if err != nil {
		// On exotic hosts with neither native nor docker this is correct.
		var ref sandboxerr.RefuseToRun
		if !errors.As(err, &ref) {
			t.Fatalf("Resolve(auto) = %v", err)
		}
		return
	}
	switch runtime.GOOS {
	case "linux":
		if name != BackendLinux && name != BackendDocker {
			t.Fatalf("Resolve(auto) = %q, want linux or docker", name)
		}
		if linuxAvailable() && name != BackendLinux {
			t.Fatalf("Resolve(auto) = %q, want linux when bwrap is available", name)
		}
	case "darwin":
		if name != BackendSeatbelt && name != BackendDocker {
			t.Fatalf("Resolve(auto) = %q, want seatbelt or docker", name)
		}
		if seatbeltAvailable() && name != BackendSeatbelt {
			t.Fatalf("Resolve(auto) = %q, want seatbelt when sandbox-exec is available", name)
		}
	case "windows":
		if name != BackendWindows && name != BackendDocker {
			t.Fatalf("Resolve(auto) = %q, want windows or docker", name)
		}
		if windowsAvailable() && name != BackendWindows {
			t.Fatalf("Resolve(auto) = %q, want windows when AppContainer is available", name)
		}
	default:
		if name != BackendDocker {
			t.Fatalf("Resolve(auto) = %q, want docker on %s", name, runtime.GOOS)
		}
	}
}

func TestResolveExplicitUnknown(t *testing.T) {
	if _, err := Resolve("plan9"); err == nil {
		t.Fatal("expected error for unknown backend")
	}
}

func TestResolveExplicitLinuxRequiresHost(t *testing.T) {
	_, err := Resolve("linux")
	if runtime.GOOS == "linux" && linuxAvailable() {
		if err != nil {
			t.Fatalf("Resolve(linux) on linux with bwrap: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatal("expected Resolve(linux) to fail on this host")
	}
}

func TestResolveExplicitSeatbeltRequiresDarwin(t *testing.T) {
	_, err := Resolve("seatbelt")
	if runtime.GOOS == "darwin" && seatbeltAvailable() {
		if err != nil {
			t.Fatalf("Resolve(seatbelt) on darwin: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatal("expected Resolve(seatbelt) to fail on this host")
	}
}

func TestResolveExplicitWindowsRequiresWindows(t *testing.T) {
	_, err := Resolve("windows")
	if runtime.GOOS == "windows" && windowsAvailable() {
		if err != nil {
			t.Fatalf("Resolve(windows) on windows: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatal("expected Resolve(windows) to fail on this host")
	}
}

func TestResolveAliases(t *testing.T) {
	if runtime.GOOS == "linux" && linuxAvailable() {
		name, err := Resolve("bwrap")
		if err != nil || name != BackendLinux {
			t.Fatalf("Resolve(bwrap) = %q, %v", name, err)
		}
	}
	if runtime.GOOS == "darwin" && seatbeltAvailable() {
		name, err := Resolve("macos")
		if err != nil || name != BackendSeatbelt {
			t.Fatalf("Resolve(macos) = %q, %v", name, err)
		}
	}
	if runtime.GOOS == "windows" && windowsAvailable() {
		if name, err := Resolve("appcontainer"); err != nil || name != BackendWindows {
			t.Fatalf("Resolve(appcontainer) = %q, %v", name, err)
		}
	}
}

func TestRefusalMessageContainsFailClosedByDesign(t *testing.T) {
	// This test ensures the "fails closed by design" framing cannot be
	// silently removed in a future edit. Every RefuseToRun error must
	// explain that Warden's refusal is intentional.
	reasons := []string{
		"Warden isn't running with administrator privileges",
		"Warden isn't running with bubblewrap (bwrap) or Docker available on this Linux host",
		"Warden isn't running with sandbox-exec or Docker available on this macOS host",
		"Docker is not installed on this host",
		"the Docker daemon is not running or not reachable on this host",
	}
	for _, reason := range reasons {
		err := sandboxerr.RefuseToRun{Reason: reason}
		msg := err.Error()
		if !strings.Contains(msg, "fails closed by design") {
			t.Errorf("RefuseToRun(%q) missing 'fails closed by design':\n%s", reason, msg)
		}
		if !strings.Contains(msg, "✗ Warden refused to start") {
			t.Errorf("RefuseToRun(%q) missing '✗ Warden refused to start':\n%s", reason, msg)
		}
	}
}
