// Package darwin implements the macOS sandbox backend using sandbox-exec
// (Seatbelt). Profile generation is pure and covered by unit tests on every
// OS; the Run path is darwin-only (see run.go).
package darwin

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/warden-sandbox/warden/internal/policy"
)

// runtimeReadPaths are always readable so a typical macOS userspace binary
// can start (dyld, frameworks, system libs). They are never writable.
var runtimeReadPaths = []string{
	"/usr",
	"/bin",
	"/sbin",
	"/System",
	"/Library/Frameworks",
	"/Library/Apple",
	"/private/var/db/dyld",
	"/dev",
	"/private/tmp",
	"/tmp",
	"/var/folders",
	"/private/var/folders",
}

// BuildSeatbeltProfile translates a policy into an SBPL (Seatbelt) profile.
// Network is deny-by-default except loopback TCP to the egress proxy bridge
// and the Unix socket used to reach the host-side proxy.
func BuildSeatbeltProfile(cmd []string, p policy.Policy, socketPath string) (string, error) {
	if len(cmd) == 0 {
		return "", fmt.Errorf("seatbelt profile: no command")
	}
	exe := cmd[0]
	if !filepath.IsAbs(exe) {
		return "", fmt.Errorf("seatbelt profile: command %q must be an absolute path", exe)
	}
	if socketPath == "" {
		return "", fmt.Errorf("seatbelt profile: proxy socket path is required")
	}
	if !filepath.IsAbs(socketPath) {
		return "", fmt.Errorf("seatbelt profile: proxy socket path %q must be absolute", socketPath)
	}

	mode := make(map[string]string)
	for _, path := range p.Filesystem.Read {
		if mode[path] == "write" {
			return "", fmt.Errorf("seatbelt profile: %q is granted as both read and write", path)
		}
		mode[path] = "read"
	}
	for _, path := range p.Filesystem.Write {
		if mode[path] == "read" {
			return "", fmt.Errorf("seatbelt profile: %q is granted as both read and write", path)
		}
		for _, base := range []string{"/usr", "/bin", "/sbin", "/System"} {
			if path == base || isUnder(path, base) {
				return "", fmt.Errorf("seatbelt profile: %q is inside the read-only runtime base %s and cannot be granted write", path, base)
			}
		}
		mode[path] = "write"
	}

	var b strings.Builder
	b.WriteString("(version 1)\n")
	b.WriteString("(deny default)\n")
	b.WriteString("; Process lifecycle\n")
	b.WriteString("(allow process*)\n")
	b.WriteString("(allow signal)\n")
	b.WriteString("(allow sysctl-read)\n")
	b.WriteString("(allow mach-lookup)\n")
	b.WriteString("(allow file-read-metadata)\n")
	b.WriteString("; Pseudo / runtime paths\n")
	for _, path := range runtimeReadPaths {
		writeSubpathAllow(&b, "file-read*", path)
	}
	// Writable scratch space for temp files.
	writeSubpathAllow(&b, "file-write*", "/tmp")
	writeSubpathAllow(&b, "file-write*", "/private/tmp")
	writeSubpathAllow(&b, "file-write*", "/var/folders")
	writeSubpathAllow(&b, "file-write*", "/private/var/folders")

	b.WriteString("; Policy filesystem grants\n")
	for _, path := range p.Filesystem.Read {
		writeSubpathAllow(&b, "file-read*", path)
	}
	for _, path := range p.Filesystem.Write {
		writeSubpathAllow(&b, "file-read*", path)
		writeSubpathAllow(&b, "file-write*", path)
	}

	// Ensure the executable's parent directory is readable.
	parent := filepath.Dir(exe)
	if mode[parent] == "" && !isRuntimePath(parent) {
		writeSubpathAllow(&b, "file-read*", parent)
	}

	b.WriteString("; Egress: only the local proxy bridge and its Unix socket\n")
	b.WriteString("(allow network-outbound (remote ip \"localhost:18080\"))\n")
	b.WriteString("(allow network-bind (local ip \"localhost:18080\"))\n")
	b.WriteString("(allow network-inbound (local ip \"localhost:18080\"))\n")
	b.WriteString("(allow network-outbound (remote unix-socket))\n")
	writeLiteralAllow(&b, "file-read*", socketPath)
	writeLiteralAllow(&b, "file-write*", socketPath)
	writeLiteralAllow(&b, "file-read*", filepath.Dir(socketPath))
	writeSubpathAllow(&b, "file-read*", filepath.Dir(socketPath))

	return b.String(), nil
}

func isRuntimePath(path string) bool {
	for _, base := range runtimeReadPaths {
		if path == base || isUnder(path, base) {
			return true
		}
	}
	return false
}

func isUnder(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func writeSubpathAllow(b *strings.Builder, action, path string) {
	fmt.Fprintf(b, "(allow %s (subpath %q))\n", action, path)
}

func writeLiteralAllow(b *strings.Builder, action, path string) {
	fmt.Fprintf(b, "(allow %s (literal %q))\n", action, path)
}
