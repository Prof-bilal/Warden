//go:build !windows

package main

const elevatedCmd = "__elevated"

// offerElevation is Windows-only (UAC); elsewhere the error is reported as-is.
func offerElevation(error) (bool, int) { return false, 0 }

func cmdElevated([]string) {}
