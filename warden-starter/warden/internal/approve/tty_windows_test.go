//go:build windows

package approve

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
	"unicode"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestWindowsConsoleApproval(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Give the probe an isolated console even when go test was launched in a terminal.
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestWindowsConsoleProbe$")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.DETACHED_PROCESS}
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("console probe failed: %v\n%s", err, output)
	}
}

func TestWindowsConsoleProbe(t *testing.T) {
	if len(os.Args) < 2 || os.Args[1] != "-test.run=^TestWindowsConsoleProbe$" {
		return
	}
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	if result, _, err := kernel.NewProc("AllocConsole").Call(); result == 0 {
		t.Fatalf("allocate console: %v", err)
	}
	defer kernel.NewProc("FreeConsole").Call()
	if err := CheckTTY(); err != nil {
		t.Fatal(err)
	}
	input, err := os.OpenFile("CONIN$", os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	write := kernel.NewProc("WriteConsoleInputW")
	prompt := NewPrompter("", 3*time.Second, nil)
	for _, test := range []struct {
		line string
		yes  bool
	}{{"no\r", false}, {"yes\r", true}} {
		keys := make([]consoleKey, 0, len(test.line))
		for _, char := range test.line {
			keys = append(keys, consoleKey{kind: 1, down: 1, repeats: 1, virtualKey: uint16(unicode.ToUpper(char)), character: uint16(char)})
		}
		var written uint32
		ok, _, err := write.Call(input.Fd(), uintptr(unsafe.Pointer(&keys[0])), uintptr(len(keys)), uintptr(unsafe.Pointer(&written)))
		if ok == 0 || int(written) != len(keys) {
			t.Fatalf("write console input: %v", err)
		}
		yes, err := prompt.Confirm("Approve disposable fixture?")
		if err != nil || yes != test.yes {
			t.Fatalf("approval = %t, %v, want %t", yes, err, test.yes)
		}
	}
}

// INPUT_RECORD with a KEY_EVENT_RECORD in the event union.
type consoleKey struct {
	kind                uint16
	_                   uint16
	down                uint32
	repeats, virtualKey uint16
	scanCode, character uint16
	controlKeys         uint32
}
