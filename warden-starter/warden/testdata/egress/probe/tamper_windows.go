//go:build windows

package main

import (
	"flag"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

func terminateParent(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	delay := durationFlag(fs, "delay", "100ms", "delay before signalling")
	target := fs.String("probe-addr", "", "optional address to dial after signalling")
	probeURL := fs.String("probe-url", "", "optional allowed URL to retry through the proxy after signalling")
	timeout := durationFlag(fs, "timeout", "3s", "timeout for post-signal probe")
	_ = fs.Parse(args)

	ppid := os.Getppid()
	emitUp(action)
	emit(result{"action": action, "ok": true, "phase": "before", "ppid": ppid, "signal": "terminate"})
	time.Sleep(*delay)
	proc, err := os.FindProcess(ppid)
	if err == nil {
		err = proc.Kill()
	}
	res := result{"action": action, "ok": err == nil, "phase": "signal", "ppid": ppid}
	if err != nil {
		res["error"] = err.Error()
		res["error_class"] = classify(err)
	}
	emit(res)
	time.Sleep(200 * time.Millisecond)
	// Windows has no signal-0 liveness probe; report unknown so the runner
	// does not overstate what the kill attempt proved.
	emit(result{"action": action, "phase": "after-parent", "ppid": ppid, "alive_unknown": true})
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
	terminateParent(action, args)
}

func doSignalParent(action string, args []string) {
	terminateParent(action, args)
}
