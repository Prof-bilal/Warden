//go:build windows

package windows

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// The loopback exemption list is the machine-wide NetworkIsolation
// AppContainer config (what `CheckNetIsolation LoopbackExempt` edits). Warden
// adds only this run's package SID and restores the previous list afterwards.
var (
	firewallAPI                               = windows.NewLazySystemDLL("FirewallAPI.dll")
	procNetworkIsolationGetAppContainerConfig = firewallAPI.NewProc("NetworkIsolationGetAppContainerConfig")
	procNetworkIsolationSetAppContainerConfig = firewallAPI.NewProc("NetworkIsolationSetAppContainerConfig")
)

// exemptLoopback adds packageSID to the loopback exemption list and returns a
// func that restores the list as it was.
func exemptLoopback(packageSID string) (func(), error) {
	pkg, err := windows.StringToSid(packageSID)
	if err != nil {
		return nil, failClose("loopback exemption package SID", err)
	}
	prev, err := loopbackExemptions()
	if err != nil {
		return nil, failClose("read loopback exemptions", err)
	}
	next := append(append([]sidAndAttributes{}, prev...), sidAndAttributes{Sid: pkg})
	if err := setLoopbackExemptions(next); err != nil {
		return nil, failClose("add loopback exemption", err)
	}
	return func() { _ = setLoopbackExemptions(prev) }, nil
}

// loopbackExemptions returns copies of the currently exempted SIDs.
func loopbackExemptions() ([]sidAndAttributes, error) {
	var count uint32
	var arr *sidAndAttributes
	if r, _, _ := procNetworkIsolationGetAppContainerConfig.Call(uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&arr))); r != 0 {
		return nil, windows.Errno(r)
	}
	if arr == nil || count == 0 {
		return nil, nil
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(arr)))
	out := make([]sidAndAttributes, 0, count)
	for _, e := range unsafe.Slice(arr, count) {
		cp, err := e.Sid.Copy()
		if err != nil {
			return nil, err
		}
		out = append(out, sidAndAttributes{Sid: cp, Attributes: e.Attributes})
	}
	return out, nil
}

func setLoopbackExemptions(list []sidAndAttributes) error {
	var p uintptr
	if len(list) > 0 {
		p = uintptr(unsafe.Pointer(&list[0]))
	}
	if r, _, _ := procNetworkIsolationSetAppContainerConfig.Call(uintptr(len(list)), p); r != 0 {
		return windows.Errno(r)
	}
	return nil
}
