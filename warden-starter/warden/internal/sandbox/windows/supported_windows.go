//go:build windows

package windows

import "golang.org/x/sys/windows"

// Supported reports whether the AppContainer sandbox backend can run on this
// host: it requires the lowbox token API, the ability to open our own process
// token, and the WFP user-mode DLL with all required exports. Detection is a
// lightweight probe; full enforcement still fails closed inside Run.
func Supported() bool {
	if err := procNtCreateLowBoxToken.Find(); err != nil {
		return false
	}
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), tokenQuery, &token); err != nil {
		return false
	}
	token.Close()
	// Verify the WFP egress layer's DLL exports are present. Without
	// this, Supported() can return true but Run() panics when the first
	// WFP proc.Call() fails to find its procedure.
	if err := wfpSupported(); err != nil {
		return false
	}
	return true
}
