package ui

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Banner is the canonical large Warden ASCII logo. Do not duplicate this
// constant elsewhere; render it via PrintBanner.
//
// Design constraints:
//   - fits in 80 columns
//   - uses only ASCII + block characters (U+2588) that render on Linux,
//     macOS, and Windows Terminal
//   - no external font required
//   - centered subtitle conveys product identity without excessive decoration
const Banner = "" +
	"██     ██  █████  ██████  ██████  ███████ ███    ██\n" +
	"██     ██ ██   ██ ██   ██ ██   ██ ██      ████   ██\n" +
	"██  █  ██ ███████ ██████  ██   ██ █████   ██ ██  ██\n" +
	"██ ███ ██ ██   ██ ██   ██ ██   ██ ██      ██  ██ ██\n" +
	" ███ ███  ██   ██ ██   ██ ██████  ███████ ██   ████"

const bannerSubtitle = "MCP SERVER SANDBOX"
const bannerTagline = "Secure execution for MCP servers"

// TerminalWidth returns the terminal width or 80 when unknown.
func TerminalWidth() int {
	if v := os.Getenv("COLUMNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	// Try to get width via env WARDEN_TERM_WIDTH for tests.
	if v := os.Getenv("WARDEN_TERM_WIDTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 80
}

// PrintBanner renders the large banner to w. When color is enabled the
// block art is rendered in cyan as structural color; the subtitle is dim.
// Falls back to plain text when color is disabled. Returns the number of
// bytes written. On narrow terminals (<60 cols) it falls back to a compact
// header so the output remains readable.
func PrintBanner(w io.Writer) (int, error) {
	if TerminalWidth() < 60 {
		// Narrow: compact header instead of wide art
		return PrintHeader(w, HeaderOptions{ShowVersion: false})
	}
	var sb strings.Builder
	lines := strings.Split(Banner, "\n")
	for _, l := range lines {
		if ColorEnabled() {
			sb.WriteString(Cyan(l))
		} else {
			sb.WriteString(l)
		}
		sb.WriteString("\n")
	}
	// Subtitle centered under the banner width (≈46 chars).
	sub := bannerSubtitle
	if ColorEnabled() {
		sub = Dim(sub)
	}
	// Banner is 46 wide; subtitle 18 chars; pad to roughly center.
	pad := ""
	if len(lines) > 0 {
		bw := len(lines[0])
		sw := len(bannerSubtitle)
		if bw > sw {
			pad = strings.Repeat(" ", (bw-sw)/2)
		}
	}
	sb.WriteString(pad + sub + "\n")
	return fmt.Fprint(w, sb.String())
}

// Header prints a reusable Warden header with optional version and platform
// information. This is smaller than the full banner and suitable for
// doctor, init, and bare invocations.
func PrintHeader(w io.Writer, opts HeaderOptions) (int, error) {
	var sb strings.Builder
	title := "WARDEN"
	if ColorEnabled() {
		title = Bold(Cyan(title))
	}
	sb.WriteString(title + "\n")
	sub := "MCP Server Sandbox Runtime"
	if ColorEnabled() {
		sub = Dim(sub)
	} else {
		// plain
	}
	sb.WriteString(sub + "\n")
	if opts.ShowVersion {
		ver := VersionString()
		plat := PlatformString()
		line := fmt.Sprintf("Version %s  %s", ver, plat)
		if ColorEnabled() {
			line = Dim(line)
		}
		sb.WriteString(line + "\n")
	}
	if opts.Extra != "" {
		sb.WriteString(opts.Extra + "\n")
	}
	return fmt.Fprint(w, sb.String())
}

// HeaderOptions controls what PrintHeader emits.
type HeaderOptions struct {
	ShowVersion bool
	Extra       string
}

// BannerWidth returns the display width of the banner's widest line.
// It counts runes, since block characters are multi-byte but single-column.
func BannerWidth() int {
	max := 0
	for _, l := range strings.Split(Banner, "\n") {
		w := len([]rune(l))
		if w > max {
			max = w
		}
	}
	return max
}

// PrintBannerWithTagline prints the large banner plus the marketing tagline
// (used for first-run and install success screens).
func PrintBannerWithTagline(w io.Writer) (int, error) {
	n1, err := PrintBanner(w)
	if err != nil {
		return n1, err
	}
	tag := bannerTagline
	if ColorEnabled() {
		tag = Dim(tag)
	}
	// Center tagline similarly.
	lines := strings.Split(Banner, "\n")
	bw := 0
	if len(lines) > 0 {
		bw = len(lines[0])
	}
	pad := ""
	if bw > len(tag) {
		// Dim-wrapped tag length differs; use raw length for padding calc
		pad = strings.Repeat(" ", (bw-len(bannerTagline))/2)
	}
	n2, err := fmt.Fprintf(w, "\n%s%s\n", pad, tag)
	return n1 + n2, err
}
