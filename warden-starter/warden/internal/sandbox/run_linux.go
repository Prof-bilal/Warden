//go:build linux

// Package sandbox selects the secure backend for the current operating
// system. Platform selection is deliberately separate from the CLI so an
// unsupported platform cannot accidentally fall back to an unrestricted run.
package sandbox

import (
	"github.com/warden-sandbox/warden/internal/policy"
	linuxbackend "github.com/warden-sandbox/warden/internal/sandbox/linux"
)

func Run(cmd []string, p policy.Policy) (int, error) { return linuxbackend.Run(cmd, p) }
