//go:build windows

package windows

import "golang.org/x/sys/windows"

// Supported reports whether the AppContainer sandbox backend can run on this
// host. WFP is an optional capability: it is needed only for policies with a
// network allowlist. A standard user can still get a complete filesystem,
// process and environment boundary with network disabled, so refusing to
// advertise the backend when WFP is unavailable would unnecessarily trigger
// UAC for ordinary offline runs.
func Supported() bool {
	if err := procNtCreateLowBoxToken.Find(); err != nil {
		return false
	}
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), tokenQuery, &token); err != nil {
		return false
	}
	token.Close()
	return true
}
