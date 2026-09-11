package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/warden-sandbox/warden/internal/sandbox"
	"github.com/warden-sandbox/warden/internal/sandbox/sandboxerr"
	"github.com/warden-sandbox/warden/internal/ui"
	"github.com/warden-sandbox/warden/internal/version"
)

func cmdDoctor(args []string) {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "warden doctor: unexpected arguments %v\n", args)
		fmt.Fprintln(os.Stderr, "usage: warden doctor")
		os.Exit(2)
	}

	// First-run welcome if this is the first meaningful invocation.
	if ui.IsTerminalWriter(os.Stderr) && !ui.IsCI() && ui.IsFirstRun() {
		_, _ = ui.PrintWelcome(os.Stderr)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, ui.Dim("──────────────────────────────────────"))
		fmt.Fprintln(os.Stderr, "")
		_ = ui.MarkFirstRun()
	} else if ui.IsTerminalWriter(os.Stderr) && !ui.IsCI() {
		_, _ = ui.PrintBanner(os.Stderr)
		fmt.Fprintln(os.Stderr, "")
	}

	// Build checks reflecting actual implementation.
	var checks []ui.Check

	// OS
	checks = append(checks, ui.Check{
		Name:   "Operating system",
		Status: "info",
		Detail: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	})
	checks = append(checks, ui.Check{
		Name:   "Warden version",
		Status: "info",
		Detail: versionString(),
	})

	// Sandbox backend
	backendName, err := sandbox.Resolve("auto")
	backendDetail := ""
	backendStatus := "ok"
	if err != nil {
		var ref sandboxerr.RefuseToRun
		if errors.As(err, &ref) {
			backendStatus = "fail"
			backendDetail = "unavailable"
		} else {
			backendStatus = "fail"
			backendDetail = err.Error()
		}
	} else {
		backendDetail = backendName
		// Enhance with binary check
		switch backendName {
		case sandbox.BackendLinux:
			if _, e := exec.LookPath("bwrap"); e != nil {
				backendStatus = "fail"
				backendDetail = "bwrap not found"
			}
		case sandbox.BackendSeatbelt:
			if _, e := exec.LookPath("sandbox-exec"); e != nil {
				backendStatus = "fail"
				backendDetail = "sandbox-exec not found"
			}
		case sandbox.BackendDocker:
			backendDetail = "docker (fallback)"
		case sandbox.BackendWindows:
			backendDetail = "AppContainer"
		}
	}
	checks = append(checks, ui.Check{
		Name:   "Sandbox backend",
		Status: backendStatus,
		Detail: backendDetail,
	})

	// Namespace support (Linux bwrap implies user namespaces)
	if runtime.GOOS == "linux" {
		if _, e := exec.LookPath("bwrap"); e == nil {
			checks = append(checks, ui.Check{Name: "Namespace support", Status: "ok", Detail: "available (bwrap)"})
		} else if backendName == sandbox.BackendDocker {
			checks = append(checks, ui.Check{Name: "Namespace support", Status: "warn", Detail: "bwrap missing, using Docker fallback", Hint: "Install bubblewrap for native isolation: sudo apt install bubblewrap"})
		} else {
			checks = append(checks, ui.Check{Name: "Namespace support", Status: "fail", Detail: "unavailable"})
		}
	} else if runtime.GOOS == "darwin" {
		if _, e := exec.LookPath("sandbox-exec"); e == nil {
			checks = append(checks, ui.Check{Name: "Namespace support", Status: "ok", Detail: "Seatbelt available"})
		} else {
			checks = append(checks, ui.Check{Name: "Namespace support", Status: "warn", Detail: "sandbox-exec missing, Docker fallback"})
		}
	} else if runtime.GOOS == "windows" {
		checks = append(checks, ui.Check{Name: "Namespace support", Status: "info", Detail: "AppContainer isolation"})
	}

	// Network proxy - always available as Go code, but detail
	checks = append(checks, ui.Check{Name: "Network proxy", Status: "ok", Detail: "available (egress allowlist)"})

	// Policy engine
	checks = append(checks, ui.Check{Name: "Policy engine", Status: "ok", Detail: "ready (YAML + validation)"})

	// Strace for audit on Linux. The native bwrap backend hard-requires
	// strace at run time (`warden run` refuses to start without it), so its
	// absence is a FAIL that makes the host NOT READY — matching what run
	// would actually do. The Docker fallback does its own sandboxing and
	// never invokes strace, so it is not required there.
	straceFails := false
	if runtime.GOOS == "linux" {
		if _, e := exec.LookPath("strace"); e == nil {
			if backendName == sandbox.BackendLinux {
				checks = append(checks, ui.Check{Name: "Audit (strace)", Status: "ok", Detail: "available"})
			} else {
				checks = append(checks, ui.Check{Name: "Audit (strace)", Status: "info", Detail: "available (not required for the docker backend)"})
			}
		} else {
			if backendName == sandbox.BackendLinux {
				straceFails = true
				checks = append(checks, ui.Check{Name: "Audit (strace)", Status: "fail", Detail: "strace not found", Hint: "warden run requires strace for file/network auditing on the native Linux backend: sudo apt install strace"})
			} else {
				checks = append(checks, ui.Check{Name: "Audit (strace)", Status: "info", Detail: "strace not found (not required for the docker backend)"})
			}
		}
	}

	// Fail-closed behavior is invariant
	checks = append(checks, ui.Check{Name: "Fail-closed", Status: "ok", Detail: "enabled (never runs unsandboxed)"})

	// Separate into two visual groups: Environment vs Security posture
	// For simplicity print all as Environment then synthesize posture checks.
	// Security posture is derived from backend readiness.
	ready := backendStatus == "ok" && !straceFails
	reason := ""
	if !ready {
		if straceFails {
			reason = "strace is required for `warden run` file/network auditing on the native Linux backend. Install it, or use --backend docker."
		} else if err != nil {
			reason = err.Error()
		} else {
			reason = "Sandbox backend unavailable. Warden will refuse to run servers without enforcement."
		}
	}

	// Security posture extra checks (filesystem, env, network)
	posture := []ui.Check{
		{Name: "Filesystem isolation", Status: mapStatus(ready), Detail: statusDetail(ready, "explicit paths only")},
		{Name: "Environment filtering", Status: mapStatus(ready), Detail: statusDetail(ready, "explicit vars only")},
		{Name: "Network policy", Status: mapStatus(ready), Detail: statusDetail(ready, "explicit hosts only")},
		{Name: "Fail-closed behavior", Status: "ok", Detail: "enforced"},
	}

	// Print using ui helper but with grouped output
	printDoctorGrouped(checks, posture, ready, reason)

	// Exit codes are machine-readable so scripts and CI can gate on them:
	//   0 = ready to run, 1 = NOT READY (backend or required tool missing).
	// Usage errors keep exit 2 above.
	if !ready {
		os.Exit(1)
	}
}

func mapStatus(ready bool) string {
	if ready {
		return "ok"
	}
	return "fail"
}
func statusDetail(ready bool, okDetail string) string {
	if ready {
		return okDetail
	}
	return "unavailable"
}

func versionString() string {
	v := version.Version
	if v == "" {
		return "dev"
	}
	return v
}

func printDoctorGrouped(envChecks, postureChecks []ui.Check, ready bool, reason string) {
	// Custom grouped printer to match spec layout:
	// WARDEN DOCTOR
	// Environment
	// ─────────────
	// Security posture
	// ─────────────
	// Status: READY
	cyan := ui.Cyan
	dim := ui.Dim
	bold := ui.Bold
	green := ui.Green
	red := ui.Red

	title := "WARDEN DOCTOR"
	if ui.ColorEnabled() {
		title = bold(cyan(title))
	}
	fmt.Fprintln(os.Stderr, title)
	fmt.Fprintln(os.Stderr, "")

	sec1 := "Environment"
	if ui.ColorEnabled() {
		sec1 = cyan(sec1)
	}
	fmt.Fprintln(os.Stderr, sec1)
	fmt.Fprintln(os.Stderr, dim("────────────────────────────────"))
	for _, c := range envChecks {
		printOneCheck(c)
	}
	fmt.Fprintln(os.Stderr, "")
	sec2 := "Security posture"
	if ui.ColorEnabled() {
		sec2 = cyan(sec2)
	}
	fmt.Fprintln(os.Stderr, sec2)
	fmt.Fprintln(os.Stderr, dim("────────────────────────────────"))
	for _, c := range postureChecks {
		printOneCheck(c)
	}
	fmt.Fprintln(os.Stderr, "")
	statusLabel := "Status: READY"
	statusDetail := "Sandbox enforcement is available. Warden will refuse to run without it."
	if !ready {
		statusLabel = "Status: NOT READY"
		if reason != "" {
			statusDetail = reason
		} else {
			statusDetail = "Sandbox backend unavailable. Warden will refuse to run servers without enforcement."
		}
		if ui.ColorEnabled() {
			statusLabel = red(bold(statusLabel))
			statusDetail = red(statusDetail)
		} else {
			statusLabel = bold(statusLabel)
		}
	} else {
		if ui.ColorEnabled() {
			statusLabel = green(bold(statusLabel))
			statusDetail = dim(statusDetail)
		}
	}
	fmt.Fprintln(os.Stderr, statusLabel)
	fmt.Fprintln(os.Stderr, statusDetail)
	if !ready {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, dim("Warden fails closed when sandboxing is unavailable."))
	}
}

func printOneCheck(c ui.Check) {
	var mark string
	name := c.Name
	switch c.Status {
	case "ok":
		mark = ui.CheckMark()
		if ui.ColorEnabled() {
			mark = ui.Green(mark)
		}
	case "fail":
		mark = ui.CrossMark()
		if ui.ColorEnabled() {
			mark = ui.Red(mark)
		}
	case "warn":
		mark = "!"
		if ui.ColorEnabled() {
			mark = ui.Yellow(mark)
		}
	default:
		mark = ui.Bullet()
		if ui.ColorEnabled() {
			mark = ui.Dim(mark)
		}
	}
	pad := ""
	if len(c.Name) < 22 {
		pad = repeat(" ", 22-len(c.Name))
	}
	detail := c.Detail
	if ui.ColorEnabled() && c.Status == "fail" {
		detail = ui.Red(detail)
	} else if detail != "" {
		detail = ui.Dim(detail)
	}
	fmt.Fprintf(os.Stderr, "%s %s%s  %s\n", mark, name, pad, detail)
	if c.Hint != "" {
		fmt.Fprintf(os.Stderr, "  %s %s\n", ui.Dim(ui.Arrow()), ui.Dim(c.Hint))
	}
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
