package ui

import (
	"fmt"
	"io"
	"runtime"
)

// Check describes one doctor diagnostic line.
type Check struct {
	Name    string
	Status  string // "ok", "warn", "fail", "info"
	Detail  string
	Hint    string
}

// DoctorReport is the structured result of a warden doctor run.
type DoctorReport struct {
	Checks []Check
	Ready  bool
	Reason string
}

// PrintDoctor renders a DoctorReport with Warden's security-oriented styling.
func PrintDoctor(w io.Writer, r DoctorReport) {
	// Header
	title := "WARDEN DOCTOR"
	if ColorEnabled() {
		title = Bold(Cyan(title))
	}
	fmt.Fprintln(w, title)
	fmt.Fprintln(w, "")

	// Environment section
	section := "Environment"
	if ColorEnabled() {
		section = Cyan(section)
	}
	fmt.Fprintln(w, section)
	fmt.Fprintln(w, Dim(repeat("─", 32)))
	for _, c := range r.Checks {
		// Filter to environment-ish checks for first section - we just print all
		// in order but add grouping hints via name.
		printCheck(w, c)
	}
	fmt.Fprintln(w, "")

	// Security posture is implicit in the checks; we surface status.
	statusLabel := "Status: READY"
	statusDetail := "Sandbox enforcement is available. Warden will refuse to run without it."
	if !r.Ready {
		statusLabel = "Status: NOT READY"
		if r.Reason != "" {
			statusDetail = r.Reason
		} else {
			statusDetail = "Sandbox backend unavailable. Warden will refuse to run servers without enforcement."
		}
		if ColorEnabled() {
			statusLabel = Red(Bold(statusLabel))
			statusDetail = Red(statusDetail)
		} else {
			statusLabel = Bold(statusLabel)
		}
	} else {
		if ColorEnabled() {
			statusLabel = Green(Bold(statusLabel))
			statusDetail = Dim(statusDetail)
		}
	}
	fmt.Fprintln(w, statusLabel)
	fmt.Fprintln(w, statusDetail)
	if !r.Ready {
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, Dim("Warden fails closed when sandboxing is unavailable."))
	}
}

func printCheck(w io.Writer, c Check) {
	var mark, name string
	switch c.Status {
	case "ok":
		mark = CheckMark()
		if ColorEnabled() {
			mark = Green(mark)
		}
		name = c.Name
	case "fail":
		mark = CrossMark()
		if ColorEnabled() {
			mark = Red(mark)
		}
		name = c.Name
	case "warn":
		mark = "!"
		if ColorEnabled() {
			mark = Yellow(mark)
		}
		name = c.Name
	default:
		mark = Bullet()
		if ColorEnabled() {
			mark = Dim(mark)
		}
		name = c.Name
	}
	// Align name to ~22 chars, detail after.
	// Use non-color length for padding? Simplify: left pad with spaces.
	pad := ""
	if len(c.Name) < 22 {
		pad = repeat(" ", 22-len(c.Name))
	}
	detail := c.Detail
	if ColorEnabled() && c.Status == "fail" {
		detail = Red(detail)
	} else if c.Detail != "" {
		detail = Dim(detail)
	}
	fmt.Fprintf(w, "%s %s%s  %s\n", mark, name, pad, detail)
	if c.Hint != "" {
		fmt.Fprintf(w, "  %s %s\n", Dim(Arrow()), Dim(c.Hint))
	}
}

func repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

// RuntimeChecks builds the standard set of environment checks based on
// the current host. This is used by the doctor command but exposed here
// so it can be tested without invoking OS primitives.
func RuntimeChecks(goos, goarch string) []Check {
	// Placeholder - real doctor fills dynamically
	return []Check{
		{Name: "Operating system", Status: "info", Detail: fmt.Sprintf("%s/%s (%s)", goos, goarch, runtime.Version())},
	}
}
