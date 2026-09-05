// Package linux implements the Linux sandbox backend using bubblewrap
// (bwrap). The sandbox is deny-by-default: the process sees only the
// runtime base (/usr, /lib64), pseudo-filesystems, and the paths granted by
// the policy.
package linux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/warden-sandbox/warden/internal/envfilter"
	"github.com/warden-sandbox/warden/internal/policy"
)

// runtimeBase are read-only mounts every sandboxed process gets, because a
// dynamically linked binary cannot even start without them. On merged-/usr
// distributions /lib64 is a symlink into /usr/lib64 and bwrap does not
// follow symlinks across bind mounts, so each base path must be mounted
// explicitly at its own location.
var runtimeBase = []string{"/usr", "/lib64"}

// BuildBwrapArgs translates a policy plus a resolved command into the
// complete bwrap argument list. It is a pure function so the mount logic
// can be unit-tested without bwrap installed.
//
// The returned slice includes everything up to the command: the caller must
// append the command and its arguments after it.
func BuildBwrapArgs(cmd []string, p policy.Policy) ([]string, error) {
	if len(cmd) == 0 {
		return nil, fmt.Errorf("bwrap args: no command")
	}

	exe := cmd[0]
	if !filepath.IsAbs(exe) {
		return nil, fmt.Errorf("bwrap args: command %q must be an absolute path", exe)
	}

	seen := make(map[string]bool)
	var args []string
	add := func(flag, src string) {
		// Under /tmp the fresh tmpfs is already mounted: give the bind an
		// explicit mountpoint first, or the tmpfs swallows it.
		if strings.HasPrefix(src, "/tmp/") {
			args = append(args, "--dir", src)
		}
		args = append(args, flag, src, src)
		seen[src] = true
	}

	// Namespace isolation: the process gets its own user/ipc/pid/net
	// namespaces. Network egress policy is M2; --unshare-net already cuts
	// the process off from the host network.
	args = append(args,
		"--unshare-user", "--unshare-ipc", "--unshare-pid", "--unshare-net",
		"--uid", "0", "--gid", "0",
	)

	// Dev and proc first: later binds under them are rare, and bwrap applies
	// mounts in order. /tmp is replaced with a fresh writable tmpfs BEFORE
	// any path under it is mounted, so a bind to /tmp/foo needs a --dir
	// mountpoint inside the tmpfs (bwrap applies mounts in order and a
	// tmpfs mounted later would hide the earlier bind).
	args = append(args, "--dev", "/dev", "--proc", "/proc", "--tmpfs", "/tmp")

	// Runtime base (read-only).
	for _, base := range runtimeBase {
		add("--ro-bind", base)
	}

	// Policy grants. A path granted both read and write is ambiguous —
	// refuse it rather than let mount order decide.
	mode := make(map[string]string)
	for _, path := range p.Filesystem.Read {
		if mode[path] == "write" {
			return nil, fmt.Errorf("bwrap args: %q is granted as both read and write", path)
		}
		mode[path] = "read"
		if !seen[path] {
			add("--ro-bind", path)
		}
	}
	for _, path := range p.Filesystem.Write {
		if mode[path] == "read" {
			return nil, fmt.Errorf("bwrap args: %q is granted as both read and write", path)
		}
		// The runtime base is always read-only; a write grant into it would
		// remount part of the system as writable and widen the sandbox.
		for _, base := range runtimeBase {
			if path == base || isUnder(path, base) {
				return nil, fmt.Errorf("bwrap args: %q is inside the read-only runtime base %s and cannot be granted write", path, base)
			}
		}
		mode[path] = "write"
		add("--bind", path)
	}

	// If the command's directory is not already mounted (e.g. a binary
	// outside /usr, like a mise-installed node), bind its parent dir so the
	// executable stays visible.
	parent := filepath.Dir(exe)
	if !seen[parent] {
		add("--ro-bind", parent)
	}

	return args, nil
}

// isUnder reports whether path is inside dir (strictly under it). Both must
// be absolute; lexical clean is applied so "./a/.." cases are handled.
func isUnder(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Run spawns the command inside a bubblewrap sandbox built from the policy.
// Stdio is passed through directly: the sandboxed process's stdin/stdout/
// stderr are the parent's, so an MCP client talking over stdio sees no
// protocol-level difference.
//
// It returns the sandboxed process's exit code, or an error if the process
// could not be started at all. A missing bwrap is an error — Warden never
// falls back to running the command unsandboxed.
func Run(cmd []string, p policy.Policy) (int, error) {
	return runWithEnv(cmd, p, os.Environ())
}

func runWithEnv(cmd []string, p policy.Policy, parentEnv []string) (int, error) {
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		return 0, fmt.Errorf("bwrap not found: %w (required for the Linux sandbox backend; see ARCHITECTURE.md)", err)
	}

	args, err := BuildBwrapArgs(cmd, p)
	if err != nil {
		return 0, fmt.Errorf("build bwrap args: %w", err)
	}

	sub := exec.Command(bwrap, append(args, cmd...)...)
	sub.Stdin = os.Stdin
	sub.Stdout = os.Stdout
	sub.Stderr = os.Stderr
	// Deny by default for the environment: only policy-allowlisted names
	// are forwarded from the parent.
	sub.Env = envfilter.Filter(parentEnv, p.EnvAllowlist())

	if err := sub.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 0, fmt.Errorf("run sandboxed process: %w", err)
	}
	return 0, nil
}
