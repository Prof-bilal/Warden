package audit

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Trace runs cmd without a sandbox while recording every file and network
// syscall that strace reports. The trace file is created outside the target's
// control, then atomically imported into logger after the target exits.
func Trace(cmd, env []string, stdin io.Reader, stdout, stderr io.Writer, logger *Logger) (int, error) {
	if len(cmd) == 0 {
		return 0, fmt.Errorf("trace: no command")
	}
	strace, err := exec.LookPath("strace")
	if err != nil {
		return 0, fmt.Errorf("strace not found: required for trace mode: %w", err)
	}
	f, err := os.CreateTemp("", "warden-trace-*.log")
	if err != nil {
		return 0, fmt.Errorf("create trace file: %w", err)
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		return 0, fmt.Errorf("close trace file: %w", err)
	}
	defer os.Remove(path)

	args := append([]string{"-f", "-qq", "-s", "4096", "-e", "trace=%file,%network", "-o", path, "--"}, cmd...)
	sub := exec.Command(strace, args...)
	sub.Env = env
	sub.Stdin = stdin
	sub.Stdout = stdout
	sub.Stderr = stderr
	runErr := sub.Run()

	f, err = os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open completed trace: %w", err)
	}
	importErr := ImportStrace(f, logger)
	closeErr := f.Close()
	if importErr != nil {
		return 0, fmt.Errorf("import trace: %w", importErr)
	}
	if closeErr != nil {
		return 0, fmt.Errorf("close completed trace: %w", closeErr)
	}
	if runErr == nil {
		return 0, nil
	}
	if exit, ok := runErr.(*exec.ExitError); ok {
		return exit.ExitCode(), nil
	}
	return 0, fmt.Errorf("run traced process: %w", runErr)
}
