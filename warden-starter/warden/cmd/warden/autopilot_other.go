//go:build !linux

package main

import "fmt"

func cmdAutopilot([]string) (int, error) {
	return 1, fmt.Errorf("Autopilot requires Linux. Use warden doctor and warden agent for native platform options")
}
