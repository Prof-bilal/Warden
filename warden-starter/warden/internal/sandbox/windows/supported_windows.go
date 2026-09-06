//go:build windows

package windows

import "golang.org/x/sys/windows"

// Supported reports whether the AppContainer sandbox backend can run on this
// host: it requires the lowbox token API and the ability to open our own
// process token. Detection is a lightweight probe; full enforcement still fails
// closed inside Run. WFP availability is checked separately at the point
// where egress filters are installed (installWFPEgress), not here, so a
// host with AppContainer but without the WFP user-mode DLL can still run
// ETW-audited sandboxes.
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
