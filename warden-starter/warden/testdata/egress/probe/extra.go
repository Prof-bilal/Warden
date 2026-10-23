package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

// doExec runs an explicit command (for example an interpreter script) and
// reports its exit status and output. It never turns a failed spawn into a
// success.
func doExec(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	timeout := durationFlag(fs, "timeout", "20s", "command timeout")
	_ = fs.Parse(args)
	rest := fs.Args()
	if len(rest) == 0 {
		emit(result{"action": action, "ok": false, "error_class": "harness_error", "error": "missing command"})
		return
	}

	emitUp(action)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, rest[0], rest[1:]...)
	out, err := cmd.CombinedOutput()
	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	res := result{
		"action":    action,
		"command":   rest,
		"exit_code": exitCode,
		"output":    truncate(string(out), 2048),
		"timed_out": ctx.Err() == context.DeadlineExceeded,
		"ok":        err == nil,
	}
	if err != nil {
		res["error"] = err.Error()
		res["error_class"] = classify(err)
	}
	emit(res)
}

// doExecSelf runs the probe binary again with different arguments and
// returns the child's structured events. Used for subprocess and helper
// binary tests.
func doExecSelf(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	timeout := durationFlag(fs, "timeout", "20s", "child timeout")
	_ = fs.Parse(args)
	rest := fs.Args()

	emitUp(action)
	self, err := os.Executable()
	if err != nil {
		emit(result{"action": action, "ok": false, "error": err.Error(), "error_class": "harness_error"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, self, rest...)
	out, err := cmd.CombinedOutput()
	childEvents := []map[string]any{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil && m["probe"] == "egress" {
			childEvents = append(childEvents, m)
		}
	}
	res := result{
		"action":       action,
		"child_args":   rest,
		"child_events": childEvents,
		"child_output": truncate(string(out), 2048),
		"ok":           err == nil,
	}
	if cmd.ProcessState != nil {
		res["exit_code"] = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		res["error"] = err.Error()
		res["error_class"] = classify(err)
	}
	emit(res)
}

// doProxyServe runs a deliberately permissive HTTP proxy inside the sandbox
// to test whether a second, in-sandbox proxy can create an escape path.
func doProxyServe(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	listen := fs.String("listen", "127.0.0.1:0", "listen address")
	duration := durationFlag(fs, "duration", "10s", "serve duration")
	_ = fs.Parse(args)

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		emit(result{"action": action, "ok": false, "error": err.Error(), "error_class": "harness_error"})
		return
	}
	emit(result{"action": action, "event": "listen", "listen": ln.Addr().String(), "pid": os.Getpid()})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.URL.Host
		if r.Method == http.MethodConnect {
			target = r.Host
			conn, derr := net.DialTimeout("tcp", target, 3*time.Second)
			if derr != nil {
				emit(result{"action": action, "event": "upstream", "target": target, "ok": false, "error": derr.Error(), "error_class": classify(derr)})
				http.Error(w, "upstream failed", http.StatusBadGateway)
				return
			}
			emit(result{"action": action, "event": "upstream", "target": target, "ok": true})
			hj, ok := w.(http.Hijacker)
			if !ok {
				conn.Close()
				return
			}
			client, buf, herr := hj.Hijack()
			if herr != nil {
				conn.Close()
				return
			}
			_, _ = buf.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
			_ = buf.Flush()
			go func() { _, _ = io.Copy(conn, client); conn.Close() }()
			_, _ = io.Copy(client, conn)
			client.Close()
			return
		}
		if target == "" {
			target = r.Host
		}
		u := r.URL.String()
		req, rerr := http.NewRequest(r.Method, u, r.Body)
		if rerr != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		out, derr := http.DefaultClient.Do(req)
		if derr != nil {
			emit(result{"action": action, "event": "upstream", "target": target, "ok": false, "error": derr.Error(), "error_class": classify(derr)})
			http.Error(w, "upstream failed", http.StatusBadGateway)
			return
		}
		emit(result{"action": action, "event": "upstream", "target": target, "ok": true, "status": out.StatusCode})
		defer out.Body.Close()
		w.WriteHeader(out.StatusCode)
		_, _ = io.Copy(io.Discard, io.LimitReader(out.Body, 1<<20))
	})
	srv := &http.Server{Handler: handler}
	go func() { _ = srv.Serve(ln) }()
	time.Sleep(*duration)
	_ = srv.Close()
}

// doChildProxy starts a proxy-serve child, tries to use it to reach a denied
// destination, and reports whether the child had upstream access.
func doChildProxy(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	denyURL := fs.String("deny-url", "", "URL to request through the child proxy")
	timeout := durationFlag(fs, "timeout", "12s", "timeout")
	_ = fs.Parse(args)

	emitUp(action)
	self, err := os.Executable()
	if err != nil {
		emit(result{"action": action, "ok": false, "error": err.Error()})
		return
	}
	child := exec.Command(self, "proxy-serve", "--listen", "127.0.0.1:0", "--duration", "6s")
	stdout, err := child.StdoutPipe()
	if err != nil {
		emit(result{"action": action, "ok": false, "error": err.Error()})
		return
	}
	child.Stderr = os.Stderr
	if err := child.Start(); err != nil {
		emit(result{"action": action, "ok": false, "error": err.Error(), "error_class": "harness_error"})
		return
	}
	defer func() { _ = child.Process.Kill() }()

	sc := bufio.NewScanner(stdout)
	deadline := time.Now().Add(*timeout)
	listenAddr := ""
	for sc.Scan() && time.Now().Before(deadline) {
		line := sc.Text()
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil && m["event"] == "listen" {
			listenAddr, _ = m["listen"].(string)
			break
		}
	}
	res := result{"action": action, "ok": true, "child_listen": listenAddr}
	if listenAddr == "" {
		res["ok"] = false
		res["error"] = "child proxy did not report a listen address"
		emit(res)
		return
	}
	u, err := url.Parse("http://" + listenAddr)
	if err != nil {
		res["ok"] = false
		res["error"] = err.Error()
		emit(res)
		return
	}
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(u)},
		Timeout:   *timeout,
	}
	if *denyURL != "" {
		resp, derr := client.Get(*denyURL)
		if derr != nil {
			res["denied_error"] = derr.Error()
			res["denied_error_class"] = classify(derr)
		} else {
			res["denied_status"] = resp.StatusCode
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
		}
	}
	_ = child.Process.Kill()
	emit(res)
}

// doEnvTamper changes proxy-related environment variables in-process and
// shows that doing so does not open an escape path.
func doEnvTamper(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	denyAddr := fs.String("deny-addr", "", "denied ip:port")
	denyURL := fs.String("deny-url", "", "denied URL")
	bogusProxy := fs.String("proxy", "127.0.0.1:1", "proxy address to point the environment at")
	timeout := durationFlag(fs, "timeout", "8s", "timeout")
	_ = fs.Parse(args)

	emitUp(action)
	_ = os.Setenv("HTTP_PROXY", "http://"+*bogusProxy)
	_ = os.Setenv("HTTPS_PROXY", "http://"+*bogusProxy)
	_ = os.Setenv("ALL_PROXY", "http://"+*bogusProxy)
	_ = os.Setenv("NO_PROXY", "*")
	res := result{"action": action, "ok": true}
	if *denyAddr != "" {
		d := net.Dialer{Timeout: *timeout}
		conn, err := d.Dial("tcp", *denyAddr)
		if err != nil {
			res["direct_error"] = err.Error()
			res["direct_error_class"] = classify(err)
		} else {
			conn.Close()
			res["direct_ok"] = true
		}
	}
	if *denyURL != "" {
		client := &http.Client{Timeout: *timeout, Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
		resp, err := client.Get(*denyURL)
		if err != nil {
			res["proxy_error"] = err.Error()
			res["proxy_error_class"] = classify(err)
		} else {
			res["proxy_status"] = resp.StatusCode
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
		}
	}
	emit(res)
}

// doWS performs a WebSocket upgrade through the configured proxy.
func doWS(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	rawURL := fs.String("url", "", "ws:// or wss:// URL")
	timeout := durationFlag(fs, "timeout", "10s", "timeout")
	ca := fs.String("ca", "", "PEM CA bundle for wss")
	serverName := fs.String("server-name", "", "TLS server name for wss")
	_ = fs.Parse(args)

	emitUp(action)
	res := result{"action": action, "ok": true, "url": *rawURL}
	u, err := url.Parse(*rawURL)
	if err != nil || u.Host == "" {
		res["ok"] = false
		res["error"] = "invalid ws url"
		res["error_class"] = "harness_error"
		emit(res)
		return
	}
	host := u.Host
	if _, _, err := net.SplitHostPort(host); err != nil {
		if u.Scheme == "wss" || u.Scheme == "https" {
			host = net.JoinHostPort(host, "443")
		} else {
			host = net.JoinHostPort(host, "80")
		}
	}
	proxy, rawProxy, err := proxyAddress()
	if err != nil {
		res["ok"] = false
		res["error"] = err.Error()
		res["error_class"] = "harness_error"
		emit(res)
		return
	}
	res["proxy"] = rawProxy
	d := net.Dialer{Timeout: *timeout}
	conn, err := d.Dial("tcp", proxy)
	if err != nil {
		res["ok"] = false
		res["error"] = err.Error()
		res["error_class"] = classify(err)
		emit(res)
		return
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(*timeout))
	// Tunnel through the proxy for both ws:// (plain) and wss:// so the
	// WebSocket transport is subject to the same allowlist as HTTP. The proxy
	// authorizes the CONNECT target host:port.
	fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", host, host)
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		res["ok"] = false
		res["error"] = err.Error()
		res["error_class"] = classify(err)
		emit(res)
		return
	}
	if resp.StatusCode != 200 {
		res["status"] = resp.StatusCode
		res["ok"] = false
		res["error_class"] = classify(fmt.Errorf("proxy %d", resp.StatusCode))
		emit(res)
		return
	}
	if u.Scheme == "wss" || u.Scheme == "https" {
		tlsCfg := &tls.Config{ServerName: u.Hostname(), InsecureSkipVerify: false}
		if *serverName != "" {
			tlsCfg.ServerName = *serverName
		}
		if *ca != "" {
			pool, perr := rootPool(*ca)
			if perr != nil {
				res["ok"] = false
				res["error"] = perr.Error()
				res["error_class"] = "harness_error"
				emit(res)
				return
			}
			tlsCfg.RootCAs = pool
		}
		tconn := tls.Client(conn, tlsCfg)
		if err := tconn.Handshake(); err != nil {
			res["ok"] = false
			res["error"] = err.Error()
			res["error_class"] = classify(err)
			emit(res)
			return
		}
		conn = tconn
	}
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef"))
	path := u.RequestURI()
	fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n", path, u.Host, key)
	br = bufio.NewReader(conn)
	resp, err = http.ReadResponse(br, &http.Request{Method: http.MethodGet})
	if err != nil {
		res["ok"] = false
		res["error"] = err.Error()
		res["error_class"] = classify(err)
		emit(res)
		return
	}
	res["status"] = resp.StatusCode
	if resp.StatusCode != http.StatusSwitchingProtocols {
		res["ok"] = false
	}
	emit(res)
}
