//go:build !linux

package main

import "flag"

// doUnshare reports honestly that direct network-namespace manipulation is a
// Linux-only probe action. It is not a PASS or a FAIL of the host backend.
func doUnshare(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	_ = fs.Parse(args)
	emitUp(action)
	emit(result{
		"action":      action,
		"ok":          false,
		"error_class": "unsupported",
		"error":       "network namespace unshare is only implemented on Linux",
	})
}
