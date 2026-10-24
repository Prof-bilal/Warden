//go:build windows

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/warden-sandbox/warden/internal/sandbox/sandboxerr"
	"github.com/warden-sandbox/warden/internal/ui"
)

// elevatedCmd is the hidden subcommand an elevated relaunch runs:
// warden __elevated <output-file> <original args...>.
const elevatedCmd = "__elevated"

// needsElevation reports whether err is a sandbox failure that running as
// administrator can fix (WFP/ETW/ACL on paths the user does not own).
func needsElevation(err error) bool {
	if err == nil || windows.GetCurrentProcessToken().IsElevated() {
		return false
	}
	var ref sandboxerr.RefuseToRun
	msg := err.Error()
	return errors.As(err, &ref) || strings.Contains(msg, "Access is denied") ||
		strings.Contains(msg, "AppContainer sandbox support is not available")
}

// offerElevation asks the user for administrator permission and, if granted,
// reruns the same warden command through UAC. It returns handled=false when
// elevation was not requested so the caller reports the original error.
// WARDEN_ELEVATE=1 skips the y/N question (UAC still asks); in a
// non-interactive session without it nothing is prompted.
func offerElevation(err error) (handled bool, code int) {
	if !needsElevation(err) {
		return false, 0
	}
	if os.Getenv("WARDEN_ELEVATE") != "1" {
		if !ui.IsTerminalWriter(os.Stdin) || ui.IsCI() {
			fmt.Fprintln(os.Stderr, "warden: this run needs administrator permission; set WARDEN_ELEVATE=1 to request it via UAC")
			return false, 0
		}
		fmt.Fprintf(os.Stderr, "%v\n\nWarden needs administrator permission for this run. Request it now? [y/N] ", err)
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			return false, 0
		}
	}
	code, rerr := relaunchElevated(os.Args[1:])
	if rerr != nil {
		fmt.Fprintf(os.Stderr, "warden: administrator permission not granted: %v\n", rerr)
		return true, 1
	}
	return true, code
}

// relaunchElevated runs `warden __elevated <tmp> args...` via UAC, waits for
// it, copies its captured output to stderr/stdout, and returns its exit code.
// The elevated process gets its own console, so stdin is not forwarded.
func relaunchElevated(args []string) (int, error) {
	self, err := os.Executable()
	if err != nil {
		return 1, err
	}
	out, err := os.CreateTemp("", "warden-elevated-*.log")
	if err != nil {
		return 1, err
	}
	outPath := out.Name()
	out.Close()
	defer os.Remove(outPath)

	params := make([]string, 0, len(args)+2)
	for _, a := range append([]string{elevatedCmd, outPath}, args...) {
		params = append(params, syscall.EscapeArg(a))
	}
	cwd, _ := os.Getwd()
	h, err := shellExecuteRunAs(self, strings.Join(params, " "), cwd)
	if err != nil {
		return 1, err
	}
	defer windows.CloseHandle(h)
	if _, err := windows.WaitForSingleObject(h, windows.INFINITE); err != nil {
		return 1, err
	}
	var exit uint32
	if err := windows.GetExitCodeProcess(h, &exit); err != nil {
		return 1, err
	}
	if data, err := os.ReadFile(outPath); err == nil {
		os.Stdout.Write(data)
	}
	return int(exit), nil
}

// cmdElevated is the elevated side: redirect stdio (including the handles a
// sandboxed child inherits) to the shared file, then dispatch the original
// command line.
func cmdElevated(args []string) {
	if len(args) < 2 {
		os.Exit(2)
	}
	f, err := os.OpenFile(args[0], os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		os.Exit(1)
	}
	_ = windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, windows.Handle(f.Fd()))
	_ = windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(f.Fd()))
	os.Stdout, os.Stderr = f, f
	os.Args = append([]string{os.Args[0]}, args[1:]...)
	main()
}

// shellExecuteInfo mirrors SHELLEXECUTEINFOW (shellapi.h), amd64 layout.
type shellExecuteInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         windows.Handle
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     windows.Handle
	lpIDList     uintptr
	lpClass      *uint16
	hkeyClass    windows.Handle
	dwHotKey     uint32
	hIcon        windows.Handle
	hProcess     windows.Handle
}

const (
	seeMaskNoCloseProcess = 0x00000040
	seeMaskNoAsync        = 0x00000100
	swHide                = 0
)

var procShellExecuteExW = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW")

func shellExecuteRunAs(file, params, dir string) (windows.Handle, error) {
	verb, _ := windows.UTF16PtrFromString("runas")
	f, _ := windows.UTF16PtrFromString(file)
	p, _ := windows.UTF16PtrFromString(params)
	d, _ := windows.UTF16PtrFromString(dir)
	info := shellExecuteInfo{fMask: seeMaskNoCloseProcess | seeMaskNoAsync, lpVerb: verb, lpFile: f, lpParameters: p, lpDirectory: d, nShow: swHide}
	info.cbSize = uint32(unsafe.Sizeof(info))
	if r, _, e := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info))); r == 0 {
		return 0, e
	}
	if info.hProcess == 0 {
		return 0, errors.New("UAC launch returned no process")
	}
	return info.hProcess, nil
}
