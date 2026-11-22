//go:build linux

package learn

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/sandbox/linux"
)

func Observe(cmd []string, workspace string) ([]audit.Event, int, string, error) {
	dir, err := os.MkdirTemp("/tmp", "warden-learn-")
	if err != nil {
		return nil, 0, "", err
	}
	defer os.RemoveAll(dir)
	if policy.Within(dir, workspace) {
		return nil, 0, "", fmt.Errorf("observation workspace must be separate from learning scratch storage")
	}
	home := filepath.Join(dir, "home")
	if err := os.Mkdir(home, 0700); err != nil {
		return nil, 0, "", err
	}
	log, err := os.CreateTemp(dir, "events-")
	if err != nil {
		return nil, 0, "", err
	}
	defer log.Close()
	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{workspace}, Write: []string{home}, ProtectSecrets: true},
		Env:        policy.Env{Allow: []string{"HOME", "PATH", "TMPDIR"}},
		Limits:     policy.Limits{MemoryMB: 512, TimeoutS: 30},
	}
	env := []string{"HOME=" + home, "TMPDIR=" + home, "PATH=/usr/bin:/bin"}
	exit, runErr := linux.Observe(cmd, p, env, audit.New(log), workspace)
	var limit *linux.LimitExceededError
	limitMessage := ""
	if runErr != nil {
		if !errors.As(runErr, &limit) {
			return nil, exit, "", runErr
		}
		limitMessage = limit.Error()
	}
	if _, err := log.Seek(0, 0); err != nil {
		return nil, exit, limitMessage, err
	}
	events, err := audit.ReadEvents(log)
	return events, exit, limitMessage, err
}
