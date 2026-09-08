package ui

import (
	"io"
	"os"
	"path/filepath"

	"github.com/warden-sandbox/warden/internal/audit"
)

// firstRunMarker is the file that records that the welcome has been shown.
// It lives alongside the audit log so we reuse the existing state directory
// without creating new persistent locations.
const firstRunMarker = "welcomed"

// IsFirstRun reports whether the welcome experience should be shown.
// It returns false for CI, non-TTY, or when the marker already exists.
func IsFirstRun() bool {
	if IsCI() {
		return false
	}
	if os.Getenv("WARDEN_NO_FIRST_RUN") != "" {
		return false
	}
	// Don't show welcome for automation: if stdout/stderr are piped and no
	// explicit TTY, skip. But still allow "warden" bare invocation in TTY.
	if !isTerminal(os.Stdout) && !isTerminal(os.Stderr) {
		// If neither is a TTY, assume automation unless WARDEN_FORCE_FIRST_RUN
		if os.Getenv("WARDEN_FORCE_FIRST_RUN") == "" {
			return false
		}
	}
	dir, err := audit.StateDir()
	if err != nil {
		return false
	}
	marker := filepath.Join(dir, firstRunMarker)
	_, err = os.Stat(marker)
	return os.IsNotExist(err)
}

// MarkFirstRun records that the welcome has been shown. It is idempotent.
func MarkFirstRun() error {
	dir, err := audit.StateDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	marker := filepath.Join(dir, firstRunMarker)
	f, err := os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	_ = f.Close()
	return nil
}

// PrintWelcome renders the first-run welcome experience to w. It is safe to
// call even when color/unicode are disabled.
func PrintWelcome(w io.Writer) (int, error) {
	n := 0
	nn, _ := PrintBannerWithTagline(w)
	n += nn
	// Body - keep spacing premium, not gimmicky.
	lines := []string{
		"",
		Bold("Welcome to Warden."),
		"",
		"Warden sandboxes MCP servers using OS-level isolation",
		"and explicit security policies.",
		"",
		Green(CheckMark() + " Policy enforcement"),
		Green(CheckMark() + " Filesystem isolation"),
		Green(CheckMark() + " Network restrictions"),
		Green(CheckMark() + " Environment filtering"),
		Green(CheckMark() + " Fail-closed execution"),
		"",
		Bold("Get started:"),
		"",
		"  " + Cyan("warden init") + "                          Create a starter policy",
		"  " + Cyan("warden run --policy policy.yaml -- <server>") + "  Run sandboxed",
		"  " + Cyan("warden doctor") + "                        Check sandbox readiness",
		"",
		Dim("Documentation: https://github.com/Prof-bilal/Warden"),
		"",
		Dim("Warden fails closed when sandboxing is unavailable."),
		"",
	}
	for _, l := range lines {
		m, _ := io.WriteString(w, l+"\n")
		n += m
	}
	return n, nil
}

// MaybeShowWelcome checks IsFirstRun, prints the welcome to w if needed, and
// marks it seen. It returns true if the welcome was shown. Errors marking the
// file are ignored so a welcome failure never blocks the CLI.
func MaybeShowWelcome(w io.Writer) bool {
	if !IsFirstRun() {
		return false
	}
	_, _ = PrintWelcome(w)
	_ = MarkFirstRun()
	return true
}
