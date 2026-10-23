//go:build !windows

package main

import (
	"flag"
	"io"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"
)

func signalParent(action string, sig syscall.Signal, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	delay := durationFlag(fs, "delay", "100ms", "delay before signalling")
	target := fs.String("probe-addr", "", "optional address to dial after signalling")
	probeURL := fs.String("probe-url", "", "optional allowed URL to retry through the proxy after signalling")
	timeout := durationFlag(fs, "timeout", "3s", "timeout for post-signal probe")
	_ = fs.Parse(args)

	ppid := os.Getppid()
	emitUp(action)
	emit(result{"action": action, "ok": true, "phase": "before", "ppid": ppid, "signal": int(sig), "signal_name": sig.String()})
	time.Sleep(*delay)
	err := syscall.Kill(ppid, sig)
	res := result{"action": action, "ok": err == nil, "phase": "signal", "ppid": ppid, "signal": int(sig)}
	if err != nil {
		res["error"] = err.Error()
		res["error_class"] = classify(err)
	}
	emit(res)
	// Give the supervisor a moment to react before observing post-signal
	// reachability. A successful dial here after the supervisor was killed is
	// still blocked-or-not evidence, not a pass by itself.
	time.Sleep(200 * time.Millisecond)
	if alive, aerr := parentAlive(ppid); aerr != nil {
		emit(result{"action": action, "phase": "after-parent", "ppid": ppid, "alive_unknown": true, "error": aerr.Error()})
	} else {
		emit(result{"action": action, "phase": "after-parent", "ppid": ppid, "parent_alive": alive})
	}
	if *target != "" {
		d := net.Dialer{Timeout: *timeout}
		conn, derr := d.Dial("tcp", *target)
		post := result{"action": action, "ok": derr == nil, "phase": "after", "target": *target}
		if derr != nil {
			post["error"] = derr.Error()
			post["error_class"] = classify(derr)
		} else {
			conn.Close()
		}
		emit(post)
	}
	if *probeURL != "" {
		post := result{"action": action, "phase": "after-url", "url": *probeURL}
		client := &http.Client{Timeout: *timeout}
		resp, herr := client.Get(*probeURL)
		if herr != nil {
			post["ok"] = false
			post["error"] = herr.Error()
			post["error_class"] = classify(herr)
		} else {
			post["ok"] = true
			post["status"] = resp.StatusCode
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
		}
		emit(post)
	}
}

func doKillParent(action string, args []string) {
	signalParent(action, syscall.SIGKILL, args)
}

func doSignalParent(action string, args []string) {
	signalParent(action, syscall.SIGTERM, args)
}

// parentAlive checks whether the signalled parent still exists. A parent that
// is PID 1 of a PID namespace ignores signals it cannot handle, including
// SIGKILL, so a successful kill(2) call alone proves nothing.
func parentAlive(ppid int) (bool, error) {
	err := syscall.Kill(ppid, 0)
	if err == nil {
		return true, nil
	}
	if err == syscall.ESRCH {
		return false, nil
	}
	return false, err
}
