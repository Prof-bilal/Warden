package main

import (
	"fmt"
	"io"
)

func printAutopilotHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  warden autopilot setup
  warden autopilot plan
  warden autopilot approve [--edit | --advisor] -- <installed agent>
  warden autopilot status
  warden autopilot run [-- <agent arguments>]
  warden autopilot revoke

Enroll from the project root. Launch requires the Linux native backend.
The default policy is read-only, offline, and excludes known secret paths.`)
}
