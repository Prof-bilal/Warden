// Package darwin implements the macOS sandbox backend using sandbox-exec
// (Seatbelt). Profile generation is pure and covered by unit tests on every
// OS; the Run path is darwin-only (see run.go).
package darwin

import (
	"fmt"
	"os"
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
	"/opt/homebrew",
	"/usr/local",
	"/private/var/db/dyld",
	"/private/var/db/timezone",
	"/Library/Preferences/Logging",
	"/System/Library/CoreServices",
	"/dev",
	"/private/tmp",
	"/tmp",
}

// exeImagePaths are the directories the dynamic loader maps executable
// images (dylibs, frameworks) from. dyld uses mmap(PROT_EXEC) for these,
// which Seatbelt classifies as file-map-executable, not file-read*. Without
// this rule a (deny default) profile aborts the target before main() runs.
// The scratch dirs are included because intermediaries live there too: the
// proxy bridge (the warden binary or a `go test` binary) and any
// build-output target are executed from TMPDIR, not just /usr or /System.
var exeImagePaths = []string{
	"/usr",
	"/bin",
	"/sbin",
	"/System",
	"/Library/Frameworks",
	"/Library/Apple/usr/lib",
	"/Library/Apple/System/Library/Frameworks",
	"/Library/Apple/System/Library/PrivateFrameworks",
	"/System/Library/Extensions",
	"/System/Library/PrivateFrameworks",
	"/System/Library/SubFrameworks",
	"/opt/homebrew",
	"/usr/local",
	"/private/tmp",
	"/tmp",
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
	// macOS puts per-user temporary files under /var/folders. Grant only the
	// current process's concrete temporary directory, never every user's tree.
	tempDir := filepath.Clean(os.TempDir())
	b.WriteString("(version 1)\n")
	b.WriteString("(deny default)\n")
	b.WriteString("; Process lifecycle\n")
	b.WriteString("(allow process-exec)\n")
	b.WriteString("(allow process-fork)\n")
	b.WriteString("(allow signal (target same-sandbox))\n")
	b.WriteString("(allow process-info* (target same-sandbox))\n")
	b.WriteString("(allow sysctl-read)\n")
	b.WriteString("(allow mach-lookup)\n")
	b.WriteString("(allow file-read-metadata)\n")
	b.WriteString("(allow ipc-posix-shm-read*)\n")
	b.WriteString("(allow ipc-posix-sem)\n")
	b.WriteString("(allow iokit-open (iokit-registry-entry-class \"RootDomainUserClient\"))\n")
	// Root and per-component metadata traversal is blanket-allowed above.
	// dyld's CacheFinder also performs a file-read-data probe of "/" itself
	// while locating the shared cache and aborts the process (SIGABRT in
	// dyld4::CacheFinder, before main) when it is denied on macOS 26 — the
	// same grant ships in Codex's proven macOS 26 platform defaults. This
	// permits reading the root directory listing but no user data.
	b.WriteString("(allow file-read* file-test-existence (literal \"/\"))\n")
	b.WriteString("; Device basics every process needs (null, zero, urandom, own tty/fds)\n")
	b.WriteString("(allow file-read* file-write-data (literal \"/dev/null\"))\n")
	b.WriteString("(allow file-read* (literal \"/dev/zero\"))\n")
	b.WriteString("(allow file-read* (literal \"/dev/urandom\"))\n")
	b.WriteString("(allow file-read* (literal \"/dev/random\"))\n")
	b.WriteString("(allow file-ioctl (regex \"^/dev/ttys[0-9]+\"))\n")
	b.WriteString("(allow file-read* file-write* (subpath \"/dev/fd\"))\n")
	// /etc is required for user/group/resolver lookups; machine config, not
	// user data. Apple's own profiles and other sandbox-exec users (mise,
	// Codex) grant this in deny-default profiles.
	b.WriteString("; /etc for resolver, passwd and group lookups\n")
	b.WriteString("(allow file-read* (subpath \"/etc\"))\n")
	b.WriteString("(allow file-read* (subpath \"/private/etc\"))\n")
	// dyld maps system images with mmap(PROT_EXEC) => file-map-executable.
	b.WriteString("; Runtime image mapping for the dynamic loader\n")
	for _, path := range exeImagePaths {
		writeSubpathAllow(&b, "file-map-executable", path)
	}
	// dyld maps the *main executable* itself via file-map-executable too
	// (it revalidates the mapped image), even when the binary is already
	// file-readable. Without this the child aborts before main() exactly
	// like a missing dylib grant.
	writeSubpathAllow(&b, "file-map-executable", exe)
	writeLiteralAllow(&b, "file-map-executable", exe)
	b.WriteString("; Pseudo / runtime paths\n")
	for _, path := range runtimeReadPaths {
		writeSubpathAllow(&b, "file-read*", path)
	}
	// Writable scratch space for temp files.
	writeSubpathAllow(&b, "file-write*", "/tmp")
	writeSubpathAllow(&b, "file-write*", "/private/tmp")
	if tempDir != "/tmp" && tempDir != "/private/tmp" {
		for _, variant := range symlinkVariants(tempDir) {
			writeSubpathAllow(&b, "file-read*", variant)
			writeSubpathAllow(&b, "file-map-executable", variant)
			writeSubpathAllow(&b, "file-write*", variant)
		}
	}

	b.WriteString("; Policy filesystem grants\n")
	for _, path := range p.Filesystem.Read {
		for _, variant := range symlinkVariants(path) {
			writeSubpathAllow(&b, "file-read*", variant)
		}
	}
	for _, path := range p.Filesystem.Write {
		for _, variant := range symlinkVariants(path) {
			writeSubpathAllow(&b, "file-read*", variant)
			writeSubpathAllow(&b, "file-write*", variant)
			// The target's own image may live inside a write grant (e.g. a
			// built binary in the project dir): mapping it is required to exec.
			writeSubpathAllow(&b, "file-map-executable", variant)
		}
	}

	// Ensure the executable's parent directory is readable and mappable.
	parent := filepath.Dir(exe)
	if mode[parent] == "" && !isRuntimePath(parent) {
		for _, variant := range symlinkVariants(parent) {
			writeSubpathAllow(&b, "file-read*", variant)
			writeSubpathAllow(&b, "file-map-executable", variant)
		}
	}
	// The Go runtime and libSystem probe dtrace at startup; a denied
	// dtracehelper open aborts dyld before main().
	b.WriteString("(allow file-read* file-write-data file-ioctl (literal \"/dev/dtracehelper\"))\n")

	b.WriteString("; Egress: only the local proxy bridge and its Unix socket\n")
	b.WriteString("(allow network-outbound (remote ip \"localhost:18080\"))\n")
	b.WriteString("(allow network-bind (local ip \"localhost:18080\"))\n")
	b.WriteString("(allow network-inbound (local ip \"localhost:18080\"))\n")
	b.WriteString("(allow network-outbound (remote unix-socket))\n")
	for _, variant := range symlinkVariants(socketPath) {
		writeLiteralAllow(&b, "file-read*", variant)
		writeLiteralAllow(&b, "file-write*", variant)
	}
	// The socket's directory: lookup/traversal only. SBPL subpath already
	// matches the directory itself, so no file-read* literal is emitted for
	// it — a read literal on a directory is at best redundant and trips
	// aborts in dyld's path validation on macOS 26.
	for _, variant := range symlinkVariants(filepath.Dir(socketPath)) {
		writeLiteralAllow(&b, "file-read-metadata", variant)
		writeSubpathAllow(&b, "file-read*", variant)
	}

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

// symlinkVariants returns the path and its /var vs /private/var and
// /tmp vs /private/tmp counterparts so Seatbelt grants match regardless of
// whether the kernel resolves the symlink to the private prefix.
func symlinkVariants(path string) []string {
	clean := filepath.Clean(path)
	variants := []string{clean}
	seen := map[string]struct{}{clean: {}}
	add := func(v string) {
		v = filepath.Clean(v)
		if _, ok := seen[v]; !ok {
			variants = append(variants, v)
			seen[v] = struct{}{}
		}
	}
	// Also try EvalSymlinks if available (best effort).
	if eval, err := filepath.EvalSymlinks(clean); err == nil {
		add(eval)
	}
	// Explicit /var <-> /private/var and /tmp <-> /private/tmp
	if strings.HasPrefix(clean, "/var/") {
		add("/private" + clean)
	} else if strings.HasPrefix(clean, "/private/var/") {
		add(strings.TrimPrefix(clean, "/private"))
	} else if clean == "/var" {
		add("/private/var")
	} else if clean == "/private/var" {
		add("/var")
	}
	if strings.HasPrefix(clean, "/tmp/") {
		add("/private" + clean)
	} else if strings.HasPrefix(clean, "/private/tmp/") {
		add(strings.TrimPrefix(clean, "/private"))
	} else if clean == "/tmp" {
		add("/private/tmp")
	} else if clean == "/private/tmp" {
		add("/tmp")
	}
	// For eval'd path, also apply prefix transforms
	for _, v := range append([]string{}, variants...) {
		if strings.HasPrefix(v, "/var/") {
			add("/private" + v)
		} else if strings.HasPrefix(v, "/private/var/") {
			add(strings.TrimPrefix(v, "/private"))
		}
		if strings.HasPrefix(v, "/tmp/") {
			add("/private" + v)
		} else if strings.HasPrefix(v, "/private/tmp/") {
			add(strings.TrimPrefix(v, "/private"))
		}
	}
	return variants
}
