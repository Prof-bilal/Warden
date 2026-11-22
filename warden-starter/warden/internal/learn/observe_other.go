//go:build !linux

package learn

import (
	"fmt"

	"github.com/warden-sandbox/warden/internal/audit"
)

func Observe(_ []string, _ string) ([]audit.Event, int, string, error) {
	return nil, 0, "", fmt.Errorf("Linux only; use warden trace / init")
}
