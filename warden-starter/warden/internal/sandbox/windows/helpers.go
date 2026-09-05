//go:build windows

package windows

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows"
)

// This file holds low-level Win32 helpers shared by the enforcement files:
// handle management, process-tree snapshots, and path conversion.

// failClose returns a fatal error for the security gate: the message must
// make clear that Warden refused to run rather than weaken enforcement.
func failClose(what string, err error) error {
	return fmt.Errorf("refusing to run: %s failed: %w (Warden never falls back to an unsandboxed process)", what, err)
}

// newUnicodeString converts a Go string to a NUL-terminated UTF-16 buffer
// for the *W Win32 APIs.
func utf16Ptr(s string) *uint16 {
	p, err := windows.UTF16PtrFromString(s)
	if err != nil {
		// Callers validate inputs in BuildPlan; a NUL here would mean the
		// policy was not validated. Fail loudly rather than silently pass
		// a truncated string to Windows.
		panic(fmt.Sprintf("windows backend: invalid string %q: %v", s, err))
	}
	return p
}

// createStringFromUTF16 converts a NUL-terminated UTF-16 buffer to a string.
func stringFromUTF16(buf []uint16) string {
	for i, c := range buf {
		if c == 0 {
			return string(utf16Decode(buf[:i]))
		}
	}
	return string(utf16Decode(buf))
}

func utf16Decode(u []uint16) []rune {
	// Mirror windows.UTF16ToString semantics without importing the package
	// for its helper only.
	out := make([]rune, 0, len(u))
	for i := 0; i < len(u); i++ {
		switch {
		case u[i] >= 0xD800 && u[i] < 0xDC00 && i+1 < len(u) && u[i+1] >= 0xDC00 && u[i+1] < 0xE000:
			out = append(out, ((rune(u[i])-0xD800)<<10|(rune(u[i+1])-0xDC00))+0x10000)
			i++
		default:
			out = append(out, rune(u[i]))
		}
	}
	return out
}

// imageNTPath converts a Win32 executable path into the NT device path form
// used by WFP's ALE_APP_ID condition: "C:\a\b.exe" becomes
// "\device\harddiskvolumeN\a\b.exe". The volume number is resolved with
// QueryDosDeviceW rather than guessed.
func imageNTPath(win32Path string) (string, error) {
	if !strings.HasPrefix(strings.ToLower(win32Path), "c:") {
		return "", fmt.Errorf("image path %q is not on the system drive", win32Path)
	}
	// Resolve the volume of the executable without requiring a drive-letter
	// API that varies by build: QueryDosDevice maps "C:" to its device.
	vol, err := dosDeviceTarget("C:")
	if err != nil {
		return "", err
	}
	rest := win32Path[2:] // "\a\b.exe"
	if !strings.HasPrefix(rest, `\`) && !strings.HasPrefix(rest, `/`) {
		rest = `\` + rest
	}
	return strings.ToLower(vol) + rest, nil
}

// dosDeviceTarget resolves a DOS drive (e.g. "C:") to its NT device name
// (e.g. "\device\harddiskvolume3") via QueryDosDeviceW.
func dosDeviceTarget(drive string) (string, error) {
	buf := make([]uint16, 1024)
	n, err := _QueryDosDeviceW(utf16Ptr(drive), &buf[0], uint32(len(buf)))
	if err != nil || n == 0 {
		return "", fmt.Errorf("QueryDosDevice(%s): %w", drive, err)
	}
	return stringFromUTF16(buf[:n]), nil
}
