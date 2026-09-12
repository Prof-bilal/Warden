// Package docker implements the Docker fallback sandbox backend.
//
// It is used when the OS-native backend is unavailable, or when the user
// passes --backend docker. The container is deny-by-default: only the
// runtime base and policy-granted paths are bind-mounted, the network
// namespace is empty except loopback (egress goes through the host proxy
// bridge), and the environment is filtered to env.allow.
package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/envfilter"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/proxy"
	"github.com/warden-sandbox/warden/internal/sandbox/sandboxerr"
)

// DefaultImage is used when WARDEN_DOCKER_IMAGE is unset. Host /usr (and
// related paths) are bind-mounted over the image root so the sandboxed
// command runs the host's binaries inside an isolated mount + netns.
const DefaultImage = "alpine:3.20"

// runtimeBase are read-only host paths every container needs so dynamically
// linked host binaries can start. Mirrored from the Linux bwrap backend.
var runtimeBase = []string{"/usr", "/lib", "/lib64", "/bin"}

// Available reports whether the Docker CLI can talk to a working daemon.
func Available() bool {
	docker, err := exec.LookPath("docker")
	if err != nil {
		return false
	}
	cmd := exec.Command(docker, "info")
	cmd.Env = os.Environ()
	// Discard output: we only care about exit status.
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// Image returns the container image Warden will use.
func Image() string {
	if v := strings.TrimSpace(os.Getenv("WARDEN_DOCKER_IMAGE")); v != "" {
		return v
	}
	return DefaultImage
}

// LimitExceededError reports an enforced policy limit.
type LimitExceededError struct {
	Kind  string
	Limit string
}

func (e *LimitExceededError) Error() string {
	return fmt.Sprintf("resource limit exceeded: %s (%s)", e.Kind, e.Limit)
}

// BuildDockerArgs translates a policy plus command into `docker run` arguments
// (everything after `docker run`). Pure for unit testing without a daemon.
//
// socketHostDir is the host directory containing the egress Unix socket
// (mounted at /.warden/host-proxy). bridgeHostPath is the warden binary used
// as the in-container proxy bridge.
func BuildDockerArgs(cmd []string, p policy.Policy, bridgeHostPath, socketHostDir, image string) ([]string, error) {
	if len(cmd) == 0 {
		return nil, fmt.Errorf("docker args: no command")
	}
	exe := cmd[0]
	if !filepath.IsAbs(exe) {
		return nil, fmt.Errorf("docker args: command %q must be an absolute path", exe)
	}
	if image == "" {
		return nil, fmt.Errorf("docker args: image is required")
	}
	if bridgeHostPath == "" || socketHostDir == "" {
		return nil, fmt.Errorf("docker args: proxy bridge path and socket directory are required")
	}

	seen := make(map[string]bool)
	var args []string
	args = append(args,
		"--rm",
		"-i",
		"--network", "none",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges=true",
		"--pids-limit", "256",
		"--ulimit", "nofile=1024:1024",
		"--read-only",
		"--tmpfs", "/tmp:rw,mode=1777,nosuid,nodev,noexec,size=64m",
		"--tmpfs", "/run:rw,mode=755",
	)
	if p.Limits.MemoryMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", p.Limits.MemoryMB))
	}

	addBind := func(src, dst, mode string) {
		args = append(args, "-v", src+":"+dst+":"+mode)
		seen[src] = true
	}

	for _, base := range runtimeBase {
		if _, err := os.Stat(base); err != nil {
			continue // skip missing paths (e.g. /lib64 on some hosts)
		}
		addBind(base, base, "ro")
	}

	mode := make(map[string]string)
	for _, path := range p.Filesystem.Read {
		if mode[path] == "write" {
			return nil, fmt.Errorf("docker args: %q is granted as both read and write", path)
		}
		mode[path] = "read"
		if !seen[path] {
			addBind(path, path, "ro")
		}
	}
	for _, path := range p.Filesystem.Write {
		if mode[path] == "read" {
			return nil, fmt.Errorf("docker args: %q is granted as both read and write", path)
		}
		for _, base := range runtimeBase {
			if path == base || isUnder(path, base) {
				return nil, fmt.Errorf("docker args: %q is inside the read-only runtime base %s and cannot be granted write", path, base)
			}
		}
		mode[path] = "write"
		addBind(path, path, "rw")
	}

	parent := filepath.Dir(exe)
	if !seen[parent] {
		if _, err := os.Stat(parent); err == nil {
			addBind(parent, parent, "ro")
		}
	}

	const bridgePath = "/.warden/proxy-bridge"
	const proxyDir = "/.warden/host-proxy"
	addBind(bridgeHostPath, bridgePath, "ro")
	addBind(socketHostDir, proxyDir, "ro")

	args = append(args, image)
	args = append(args, bridgePath, "__proxy-bridge", "--socket", proxyDir+"/egress.sock", "--listen", "127.0.0.1:18080", "--")
	args = append(args, cmd...)
	return args, nil
}

func isUnder(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Run starts cmd inside a Docker container built from the policy.
func Run(cmd []string, p policy.Policy) (int, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return 0, sandboxerr.RefuseToRun{Reason: "Docker is not installed on this host"}
	}
	if !Available() {
		return 0, sandboxerr.RefuseToRun{Reason: "the Docker daemon is not running or not reachable on this host"}
	}
	logFile, _, err := audit.OpenDefault()
	if err != nil {
		return 0, fmt.Errorf("open audit log: %w", err)
	}
	defer logFile.Close()
	return runWithEnvAndAudit(cmd, p, os.Environ(), audit.New(logFile), nil)
}

// RunWithApproval starts cmd like Run with interactive approval mode for
// network requests: blocked hosts prompt on the terminal and approved ones
// apply live via the egress proxy. The container has no live file-access
// signal comparable to Linux strace, so filesystem stays hard-deny (use
// `warden trace` + `warden init` to widen it). A nil or disabled cfg behaves
// like Run.
func RunWithApproval(cmd []string, p policy.Policy, cfg *approve.Config) (int, error) {
	if cfg == nil || !cfg.Enabled {
		return Run(cmd, p)
	}
	if err := cfg.Validate(); err != nil {
		return 0, err
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return 0, sandboxerr.RefuseToRun{Reason: "Docker is not installed on this host"}
	}
	if !Available() {
		return 0, sandboxerr.RefuseToRun{Reason: "the Docker daemon is not running or not reachable on this host"}
	}
	logFile, _, err := audit.OpenDefault()
	if err != nil {
		return 0, fmt.Errorf("open audit log: %w", err)
	}
	defer logFile.Close()
	return runWithEnvAndAudit(cmd, p, os.Environ(), audit.New(logFile), cfg)
}

func runWithEnvAndAudit(cmd []string, p policy.Policy, parentEnv []string, logger *audit.Logger, approval *approve.Config) (int, error) {
	dockerBin, err := exec.LookPath("docker")
	if err != nil {
		return 0, sandboxerr.RefuseToRun{Reason: "Docker is not installed on this host"}
	}
	image := Image()
	if err := ensureImage(dockerBin, image); err != nil {
		return 0, err
	}

	eg, err := proxy.Start(p.Network.Allow, logger)
	if err != nil {
		return 0, fmt.Errorf("start egress proxy: %w", err)
	}
	defer eg.Close()

	if approval != nil && approval.Enabled {
		eg.SetApprover(approve.NewPrompter(approval.PolicyPath, approval.Timeout, logger).NetworkApprover())
	}

	bridgeExe, err := resolveBridgeExecutable()
	if err != nil {
		return 0, err
	}

	args, err := BuildDockerArgs(cmd, p, bridgeExe, filepath.Dir(eg.SocketPath()), image)
	if err != nil {
		return 0, fmt.Errorf("build docker args: %w", err)
	}

	env := envfilter.Filter(parentEnv, p.EnvAllowlist())
	runArgs := injectEnvFlags(append([]string{"run"}, args...), env, []string{
		"HTTP_PROXY=http://127.0.0.1:18080",
		"HTTPS_PROXY=http://127.0.0.1:18080",
		"ALL_PROXY=http://127.0.0.1:18080",
		"NO_PROXY=",
	})
	sub := exec.Command(dockerBin, runArgs...)
	sub.Stdin = os.Stdin
	sub.Stdout = os.Stdout
	sub.Stderr = os.Stderr
	setSandboxProcAttr(sub)
	// docker CLI itself still needs the host environment (e.g. DOCKER_HOST).
	sub.Env = os.Environ()

	if err := sub.Start(); err != nil {
		return 0, fmt.Errorf("start sandboxed process: %w", err)
	}
	runErr, limitErr := waitWithTimeout(sub, p.Limits.TimeoutS)
	if limitErr != nil {
		if logger != nil {
			_ = logger.Log(audit.Event{Type: "limit", Action: "terminate", Resource: limitErr.Limit, Allowed: false, Reason: limitErr.Error()})
		}
		return 0, limitErr
	}
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 0, fmt.Errorf("run sandboxed process: %w", runErr)
	}
	return 0, nil
}

// resolveBridgeExecutable returns a Linux ELF warden binary that can run
// inside the container as the proxy bridge. On Linux hosts the current
// executable works. On other hosts (e.g. macOS talking to Docker Desktop's
// Linux VM), the Darwin binary cannot execute in the container — callers
// must set WARDEN_DOCKER_BRIDGE to a Linux-built warden.
func resolveBridgeExecutable() (string, error) {
	if v := strings.TrimSpace(os.Getenv("WARDEN_DOCKER_BRIDGE")); v != "" {
		path, err := filepath.EvalSymlinks(v)
		if err != nil {
			return "", fmt.Errorf("resolve WARDEN_DOCKER_BRIDGE: %w", err)
		}
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("WARDEN_DOCKER_BRIDGE %q: %w", v, err)
		}
		return path, nil
	}
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("docker backend on %s needs a Linux-built warden binary for the in-container proxy bridge; set WARDEN_DOCKER_BRIDGE to that path (macOS prefers the Seatbelt backend when sandbox-exec is available)", runtime.GOOS)
	}
	bridgeExe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate proxy bridge executable: %w", err)
	}
	bridgeExe, err = filepath.EvalSymlinks(bridgeExe)
	if err != nil {
		return "", fmt.Errorf("resolve proxy bridge executable: %w", err)
	}
	return bridgeExe, nil
}

// injectEnvFlags inserts `docker run -e NAME=VALUE` flags after "run".
func injectEnvFlags(runArgs []string, filtered, proxyEnv []string) []string {
	// runArgs is ["run", ...flags..., image, ...]
	if len(runArgs) == 0 || runArgs[0] != "run" {
		return runArgs
	}
	var envFlags []string
	for _, kv := range append(append([]string{}, filtered...), proxyEnv...) {
		envFlags = append(envFlags, "-e", kv)
	}
	out := make([]string, 0, len(runArgs)+len(envFlags))
	out = append(out, "run")
	out = append(out, envFlags...)
	out = append(out, runArgs[1:]...)
	return out
}

func ensureImage(dockerBin, image string) error {
	inspect := exec.Command(dockerBin, "image", "inspect", image)
	inspect.Stdout = nil
	inspect.Stderr = nil
	if inspect.Run() == nil {
		return nil
	}
	return fmt.Errorf("docker image %q is not available locally; pull it or set WARDEN_DOCKER_IMAGE to an image you already have", image)
}

func waitWithTimeout(cmd *exec.Cmd, timeoutS int) (error, *LimitExceededError) {
	if timeoutS <= 0 {
		return cmd.Wait(), nil
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	timer := time.NewTimer(time.Duration(timeoutS) * time.Second)
	defer timer.Stop()
	select {
	case err := <-done:
		return err, nil
	case <-timer.C:
		terminateSandbox(cmd)
		select {
		case err := <-done:
			return err, &LimitExceededError{Kind: "wall-clock timeout", Limit: strconv.Itoa(timeoutS) + "s"}
		case <-time.After(750 * time.Millisecond):
			killSandbox(cmd)
			err := <-done
			return err, &LimitExceededError{Kind: "wall-clock timeout", Limit: strconv.Itoa(timeoutS) + "s"}
		}
	}
}
