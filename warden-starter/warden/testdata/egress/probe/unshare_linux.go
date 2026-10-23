//go:build linux

package main

import (
	"flag"
	"net"
	"syscall"
	"time"
)

// doUnshare attempts to create a new network namespace from inside the
// sandbox. A hardened backend should make this fail; if it succeeds, the new
// namespace has no route to the bridge and therefore still fails closed.
func doUnshare(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	addr := fs.String("addr", "", "optional address to probe after unshare")
	timeout := durationFlag(fs, "timeout", "5s", "timeout")
	_ = fs.Parse(args)

	emitUp(action)
	res := result{"action": action, "ok": true}
	err := syscall.Unshare(syscall.CLONE_NEWNET)
	if err != nil {
		res["unshare_ok"] = false
		res["error"] = err.Error()
		res["error_class"] = classify(err)
	} else {
		res["unshare_ok"] = true
	}
	if *addr != "" {
		d := net.Dialer{Timeout: *timeout}
		start := time.Now()
		conn, derr := d.Dial("tcp", *addr)
		res["post_unshare_elapsed_ms"] = time.Since(start).Milliseconds()
		if derr != nil {
			res["post_unshare_ok"] = false
			res["post_unshare_error"] = derr.Error()
			res["post_unshare_error_class"] = classify(derr)
		} else {
			conn.Close()
			res["post_unshare_ok"] = true
		}
	}
	emit(res)
}
