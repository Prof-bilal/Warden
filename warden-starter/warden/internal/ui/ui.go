// Package ui provides Warden's terminal identity: colors, symbols,
// banner, and capability detection. It is the single place for ANSI
// handling so the rest of the CLI never hard-codes escape sequences.
package ui

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/warden-sandbox/warden/internal/version"
)

// ANSI escape codes. Only used when ColorEnabled reports true.
const (
	ansiReset = "\x1b[0m"
	ansiBold  = "\x1b[1m"
	ansiDim   = "\x1b[2m"
	ansiRed   = "\x1b[31m"
	ansiGreen = "\x1b[32m"
	ansiCyan  = "\x1b[36m"
	ansiYellow = "\x1b[33m"
)

// ColorEnabled reports whether ANSI color should be emitted. It respects
// NO_COLOR (https://no-color.org), FORCE_COLOR, CLICOLOR_FORCE, and
// TERM=dumb, and falls back to TTY detection.
func ColorEnabled() bool {
	if v := os.Getenv("NO_COLOR"); v != "" {
		return false
	}
	if v := os.Getenv("FORCE_COLOR"); v != "" && v != "0" {
		return true
	}
	if v := os.Getenv("CLICOLOR_FORCE"); v != "" && v != "0" {
		return true
	}
	if term := os.Getenv("TERM"); term == "dumb" {
		return false
	}
	// Explicit opt-out.
	if v := os.Getenv("WARDEN_NO_COLOR"); v != "" {
		return false
	}
	// Check if stderr is a terminal. For library callers that pass explicit
	// writers we still gate on this, because piped output should not contain
	// escapes unless forced.
	return isTerminal(os.Stderr)
}

// isTerminal reports whether f is a character device (TTY). This matches the
// usual "isatty" check without importing golang.org/x/term.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// IsTerminalWriter reports whether w is a terminal file descriptor.
func IsTerminalWriter(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return isTerminal(f)
	}
	return false
}

// IsCI reports whether the process appears to run in CI.
func IsCI() bool {
	if v := os.Getenv("CI"); v != "" && v != "0" && strings.ToLower(v) != "false" {
		return true
	}
	for _, k := range []string{"GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "TF_BUILD", "CIRCLECI", "BUILDKITE", "TEAMCITY_VERSION"} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}

// IsNonInteractive reports whether output should avoid animations.
func IsNonInteractive(w io.Writer) bool {
	if IsCI() {
		return true
	}
	if os.Getenv("WARDEN_NON_INTERACTIVE") != "" {
		return true
	}
	if !IsTerminalWriter(w) {
		return true
	}
	if !ColorEnabled() && os.Getenv("FORCE_COLOR") == "" {
		// If color is disabled and not forced, assume non-interactive for
		// spinner purposes, but still allow plain static output.
		// We treat this as non-interactive for animation only.
		return false
	}
	return false
}

// SupportsUnicode reports whether the terminal likely supports Unicode box
// and check marks. Controlled by LANG/LC_ALL and explicit env.
func SupportsUnicode() bool {
	if v := os.Getenv("WARDEN_NO_UNICODE"); v != "" {
		return false
	}
	if v := os.Getenv("NO_UNICODE"); v != "" {
		return false
	}
	term := os.Getenv("TERM")
	if term == "dumb" {
		return false
	}
	// Windows legacy console without UTF-8 codepage often mangles block chars.
	// On Windows we still allow Unicode when WT_SESSION or TERM_PROGRAM set.
	if runtime.GOOS == "windows" {
		if os.Getenv("WT_SESSION") != "" || os.Getenv("TERM_PROGRAM") != "" {
			return true
		}
		// Default to true on modern Windows Terminal; fallback is harmless.
		return true
	}
	// Check locale hints.
	for _, k := range []string{"LANG", "LC_ALL", "LC_CTYPE"} {
		v := strings.ToLower(os.Getenv(k))
		if strings.Contains(v, "utf-8") || strings.Contains(v, "utf8") {
			return true
		}
	}
	// If no locale hint, assume UTF-8 on Linux/macOS modern terminals.
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		return true
	}
	return true
}

// colorize wraps s in ANSI codes when color is enabled.
func colorize(code, s string) string {
	if !ColorEnabled() {
		return s
	}
	return code + s + ansiReset
}

// Green renders s in Warden green (success, allowed, sandbox active).
func Green(s string) string { return colorize(ansiGreen, s) }

// Red renders s in Warden red (blocked, denied, error).
func Red(s string) string { return colorize(ansiRed, s) }

// Cyan renders s in muted cyan (structural, paths, headings).
func Cyan(s string) string { return colorize(ansiCyan, s) }

// Dim renders s in dim (secondary information).
func Dim(s string) string { return colorize(ansiDim, s) }

// Bold renders s in bold.
func Bold(s string) string { return colorize(ansiBold, s) }

// Yellow renders s in yellow (warnings).
func Yellow(s string) string { return colorize(ansiYellow, s) }

// Symbols with graceful degradation.

func CheckMark() string {
	if SupportsUnicode() {
		return "✓"
	}
	return "OK"
}

func CrossMark() string {
	if SupportsUnicode() {
		return "✗"
	}
	return "x"
}

func Bullet() string {
	if SupportsUnicode() {
		return "•"
	}
	return "-"
}

func Arrow() string {
	if SupportsUnicode() {
		return "→"
	}
	return "->"
}

// Fprintln helpers that reduce call-site verbosity.

func FprintCheck(w io.Writer, msg string) {
	fmt.Fprintf(w, "%s %s\n", Green(CheckMark()), msg)
}

func FprintCross(w io.Writer, msg string) {
	fmt.Fprintf(w, "%s %s\n", Red(CrossMark()), msg)
}

func FprintDim(w io.Writer, msg string) {
	fmt.Fprintln(w, Dim(msg))
}

// VersionString returns the stamped version or "dev".
func VersionString() string {
	v := version.Version
	if v == "" {
		return "dev"
	}
	return v
}

// PlatformString returns a concise platform descriptor.
func PlatformString() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}

// RuntimeInfo returns version plus platform for header display.
func RuntimeInfo() string {
	return fmt.Sprintf("v%s %s", VersionString(), PlatformString())
}
