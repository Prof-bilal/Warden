//go:build !linux && !darwin && !windows

package main

import "fmt"

func cmdAutopilot([]string) (int, error) {
	return 1, fmt.Errorf("Autopilot requires Linux, macOS, or Windows. Use warden doctor and warden agent for native platform options")
}
