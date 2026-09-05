//go:build linux

package linux

import (
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/warden-sandbox/warden/internal/policy"
)

func TestLimitHelper(t *testing.T) {
	if os.Getenv("WARDEN_LIMIT_HELPER") != "1" {
		return
	}
	mode := os.Args[len(os.Args)-1]
	switch mode {
	case "sleep":
		time.Sleep(10 * time.Second)
	case "memory":
		data := make([]byte, 64*1024*1024)
		for i := range data {
			data[i] = 1
		}
		runtime.KeepAlive(data)
		time.Sleep(10 * time.Second)
	}
	os.Exit(0)
}

func TestWaitWithLimitsTimeout(t *testing.T) {
	err, breach := runLimitHelper(t, "sleep", policy.Limits{TimeoutS: 1})
	if breach == nil || breach.Kind != "wall-clock timeout" {
		t.Fatalf("breach = %#v, wait error = %v", breach, err)
	}
}

func TestWaitWithLimitsMemory(t *testing.T) {
	err, breach := runLimitHelper(t, "memory", policy.Limits{MemoryMB: 8})
	if breach == nil || breach.Kind != "memory" {
		t.Fatalf("breach = %#v, wait error = %v", breach, err)
	}
}

func runLimitHelper(t *testing.T, mode string, limits policy.Limits) (error, *LimitExceededError) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestLimitHelper", "--", mode)
	cmd.Env = append(os.Environ(), "WARDEN_LIMIT_HELPER=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return waitWithLimits(cmd, limits)
}
