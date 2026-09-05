//go:build !windows

package windows

// Supported reports whether the AppContainer backend is available. On every
// non-Windows OS there is no Win32 API surface, so it is always false and the
// backend resolver never selects windows here.
func Supported() bool {
	return false
}
