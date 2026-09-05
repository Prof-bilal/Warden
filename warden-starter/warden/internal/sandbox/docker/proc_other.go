//go:build !unix

package docker

import (
	"os/exec"
)

func setSandboxProcAttr(cmd *exec.Cmd) {}

func terminateSandbox(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func killSandbox(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
