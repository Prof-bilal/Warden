//go:build windows

package mcpbridge

import "os/exec"

func configureProcess(cmd *exec.Cmd) { cmd.Cancel = func() error { terminateProcess(cmd); return nil } }
func terminateProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
