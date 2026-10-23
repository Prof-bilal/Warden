// Command probe is the sandboxed target used by the egress integrity matrix.
//
// It is deliberately dependency-free and builds as a static binary so it can
// run inside every Warden backend (bubblewrap, Seatbelt, Docker, AppContainer)
// without needing host libraries.
//
// Every action writes one JSON object per line to stdout. The runner treats a
// missing PROBE_UP marker as a harness failure, never as a security PASS.
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

type result map[string]any

func emit(r result) {
	r["probe"] = "egress"
	r["time"] = time.Now().UTC().Format(time.RFC3339Nano)
	enc := json.NewEncoder(os.Stdout)
	_ = enc.Encode(r)
}

func emitUp(action string) {
	emit(result{"action": action, "marker": "PROBE_UP", "ok": true, "pid": os.Getpid(), "ppid": os.Getppid()})
}

// classify turns transport errors into stable evidence classes. It never
// turns an incomplete observation into a success.
func classify(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "403") || strings.Contains(msg, "forbidden"):
		return "proxy_403"
	case strings.Contains(msg, "407"):
		return "proxy_407"
	case strings.Contains(msg, "502") || strings.Contains(msg, "bad gateway"):
		return "proxy_502"
	case strings.Contains(msg, "connection refused"):
		return "econnrefused"
	case strings.Contains(msg, "network is unreachable") || strings.Contains(msg, "no route to host"):
		return "enetunreach"
	case strings.Contains(msg, "operation not permitted"):
		return "eperm"
	case strings.Contains(msg, "permission denied"):
		return "eacces"
	case strings.Contains(msg, "connection reset"):
		return "econnreset"
	case strings.Contains(msg, "certificate") || strings.Contains(msg, "x509") || strings.Contains(msg, "tls:"):
		return "tls_error"
	case strings.Contains(msg, "no such host") || strings.Contains(msg, "server misbehaving") || strings.Contains(msg, "dns"):
		return "dns_error"
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded"):
		return "timeout"
	default:
		return "error"
	}
}

func durationFlag(fs *flag.FlagSet, name, def, usage string) *time.Duration {
	d, _ := time.ParseDuration(def)
	return fs.Duration(name, d, usage)
}

func tlsVersion(s string) uint16 {
	switch strings.TrimSpace(s) {
	case "1.0", "tls1.0", "TLS1.0":
		return tls.VersionTLS10
	case "1.1", "tls1.1", "TLS1.1":
		return tls.VersionTLS11
	case "1.2", "tls1.2", "TLS1.2":
		return tls.VersionTLS12
	case "1.3", "tls1.3", "TLS1.3":
		return tls.VersionTLS13
	default:
		return 0
	}
}

func rootPool(path string) (*x509.CertPool, error) {
	if path == "" {
		return nil, nil
	}
	pem, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, errors.New("no certificates found in CA file")
	}
	return pool, nil
}

func newHTTPClient(ca, serverName, minV, maxV string, insecure, forceH2, noProxy, proxyFromAll bool) (*http.Client, error) {
	pool, err := rootPool(ca)
	if err != nil {
		return nil, err
	}
	tlsCfg := &tls.Config{
		RootCAs:            pool,
		ServerName:         serverName,
		InsecureSkipVerify: insecure,
	}
	if v := tlsVersion(minV); v != 0 {
		tlsCfg.MinVersion = v
	}
	if v := tlsVersion(maxV); v != 0 {
		tlsCfg.MaxVersion = v
	}
	tr := &http.Transport{
		TLSClientConfig:    tlsCfg,
		ForceAttemptHTTP2:  forceH2,
		DisableCompression: true,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 5 * time.Second,
		}).DialContext,
	}
	if noProxy {
		tr.Proxy = nil
	} else if proxyFromAll {
		if raw := strings.TrimSpace(os.Getenv("ALL_PROXY")); raw != "" {
			u, err := url.Parse(raw)
			if err != nil {
				return nil, fmt.Errorf("parse ALL_PROXY: %w", err)
			}
			tr.Proxy = http.ProxyURL(u)
		}
	} else {
		tr.Proxy = http.ProxyFromEnvironment
	}
	return &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		},
	}, nil
}

func doHTTP(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	rawURL := fs.String("url", "", "request URL")
	method := fs.String("method", "GET", "HTTP method")
	body := fs.String("body", "", "request body")
	bodyBytes := fs.Int("body-bytes", 0, "generate a body of this many bytes")
	timeout := durationFlag(fs, "timeout", "15s", "request timeout")
	ca := fs.String("ca", "", "PEM CA bundle")
	serverName := fs.String("server-name", "", "TLS server name")
	tlsMin := fs.String("tls-min", "", "minimum TLS version (1.0..1.3)")
	tlsMax := fs.String("tls-max", "", "maximum TLS version (1.0..1.3)")
	insecure := fs.Bool("insecure", false, "skip TLS verification")
	forceH2 := fs.Bool("http2", false, "attempt HTTP/2")
	noProxy := fs.Bool("no-proxy", false, "do not use a proxy")
	proxyFromAll := fs.Bool("proxy-from-all", false, "use ALL_PROXY (curl-style) instead of Go's env lookup")
	chunked := fs.Bool("chunked", false, "send an unknown-length chunked body")
	stream := fs.Bool("stream", false, "read a streaming response")
	header := stringList{}
	fs.Var(&header, "header", "extra header (repeatable)")
	_ = fs.Parse(args)

	emitUp(action)
	if *rawURL == "" {
		emit(result{"action": action, "ok": false, "error_class": "harness_error", "error": "missing --url"})
		return
	}
	client, err := newHTTPClient(*ca, *serverName, *tlsMin, *tlsMax, *insecure, *forceH2, *noProxy, *proxyFromAll)
	if err != nil {
		emit(result{"action": action, "ok": false, "error_class": "harness_error", "error": err.Error()})
		return
	}
	client.Timeout = *timeout
	defer client.CloseIdleConnections()

	var rdr io.Reader
	if *body != "" {
		rdr = strings.NewReader(*body)
	} else if *bodyBytes > 0 {
		rdr = strings.NewReader(strings.Repeat("B", *bodyBytes))
	}
	req, err := http.NewRequest(*method, *rawURL, rdr)
	if err != nil {
		emit(result{"action": action, "ok": false, "error_class": "harness_error", "error": err.Error()})
		return
	}
	if *chunked {
		req.ContentLength = -1
	}
	for _, h := range header {
		k, v, ok := strings.Cut(h, ":")
		if ok {
			req.Header.Add(strings.TrimSpace(k), strings.TrimSpace(v))
		}
	}
	res := result{"action": action, "ok": true, "url": *rawURL, "method": *method}
	start := time.Now()
	resp, err := client.Do(req)
	res["elapsed_ms"] = time.Since(start).Milliseconds()
	if err != nil {
		res["ok"] = false
		res["error"] = err.Error()
		res["error_class"] = classify(err)
		emit(res)
		return
	}
	defer resp.Body.Close()
	res["status"] = resp.StatusCode
	res["proto"] = resp.Proto
	if resp.TLS != nil {
		res["tls_version"] = resp.TLS.Version
		res["tls_server_name"] = resp.TLS.ServerName
		res["tls_cipher"] = tls.CipherSuiteName(resp.TLS.CipherSuite)
	}
	if resp.Request != nil && resp.Request.URL != nil {
		res["final_url"] = resp.Request.URL.String()
	}
	if *stream {
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		lines := 0
		for sc.Scan() {
			lines++
			if lines <= 3 {
				res[fmt.Sprintf("event_%d", lines)] = sc.Text()
			}
		}
		res["stream_lines"] = lines
		if err := sc.Err(); err != nil {
			res["stream_error"] = err.Error()
			res["stream_error_class"] = classify(err)
		}
		emit(res)
		return
	}
	limited := io.LimitReader(resp.Body, 1<<20)
	b, readErr := io.ReadAll(limited)
	res["body_len"] = len(b)
	res["body_prefix"] = truncate(string(b), 512)
	if readErr != nil {
		res["ok"] = false
		res["error"] = readErr.Error()
		res["error_class"] = classify(readErr)
	}
	emit(res)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func doRaw(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	network := fs.String("network", "tcp", "tcp, tcp6, udp, udp6, unix")
	addr := fs.String("addr", "", "destination address")
	payload := fs.String("payload", "EGRESS-PROBE", "payload")
	timeout := durationFlag(fs, "timeout", "5s", "dial/read timeout")
	read := fs.Bool("read", true, "read a response")
	_ = fs.Parse(args)

	emitUp(action)
	res := result{"action": action, "ok": true, "network": *network, "addr": *addr}
	if *addr == "" {
		res["ok"] = false
		res["error_class"] = "harness_error"
		res["error"] = "missing --addr"
		emit(res)
		return
	}
	d := net.Dialer{Timeout: *timeout}
	start := time.Now()
	conn, err := d.Dial(*network, *addr)
	res["elapsed_ms"] = time.Since(start).Milliseconds()
	if err != nil {
		res["ok"] = false
		res["error"] = err.Error()
		res["error_class"] = classify(err)
		emit(res)
		return
	}
	defer conn.Close()
	res["local"] = conn.LocalAddr().String()
	res["remote"] = conn.RemoteAddr().String()
	if *payload != "" {
		_ = conn.SetWriteDeadline(time.Now().Add(*timeout))
		n, werr := io.WriteString(conn, *payload)
		res["write_bytes"] = n
		if werr != nil {
			res["write_error"] = werr.Error()
			res["write_error_class"] = classify(werr)
		}
	}
	if *read {
		_ = conn.SetReadDeadline(time.Now().Add(*timeout))
		buf := make([]byte, 4096)
		n, rerr := conn.Read(buf)
		res["read_bytes"] = n
		res["read_prefix"] = truncate(string(buf[:n]), 256)
		if rerr != nil {
			res["read_error"] = rerr.Error()
			res["read_error_class"] = classify(rerr)
		}
	}
	emit(res)
}

func doDNS(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	server := fs.String("server", "", "DNS server ip:port")
	name := fs.String("name", "", "name to resolve")
	network := fs.String("network", "udp", "udp or tcp")
	timeout := durationFlag(fs, "timeout", "5s", "query timeout")
	_ = fs.Parse(args)

	emitUp(action)
	res := result{"action": action, "ok": true, "name": *name, "server": *server, "network": *network}
	if *name == "" {
		res["ok"] = false
		res["error_class"] = "harness_error"
		res["error"] = "missing --name"
		emit(res)
		return
	}
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, address string) (net.Conn, error) {
			// Override the dial target so the probe can use an explicit
			// resolver without /etc/resolv.conf inside the sandbox.
			target := *server
			if target == "" {
				target = address
			}
			d := net.Dialer{Timeout: *timeout}
			return d.DialContext(ctx, *network, target)
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	start := time.Now()
	ips, err := r.LookupHost(ctx, *name)
	res["elapsed_ms"] = time.Since(start).Milliseconds()
	if err != nil {
		res["ok"] = false
		res["error"] = err.Error()
		res["error_class"] = classify(err)
		emit(res)
		return
	}
	res["ips"] = ips
	emit(res)
}

func proxyAddress() (string, string, error) {
	raw := strings.TrimSpace(os.Getenv("HTTP_PROXY"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("HTTPS_PROXY"))
	}
	if raw == "" {
		return "", "", errors.New("no HTTP_PROXY/HTTPS_PROXY in environment")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", err
	}
	return u.Host, raw, nil
}

func doConnect(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	target := fs.String("target", "", "CONNECT target host:port")
	timeout := durationFlag(fs, "timeout", "8s", "timeout")
	_ = fs.Parse(args)

	emitUp(action)
	res := result{"action": action, "ok": true, "target": *target}
	proxy, rawProxy, err := proxyAddress()
	if err != nil {
		res["ok"] = false
		res["error_class"] = "harness_error"
		res["error"] = err.Error()
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
	fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Connection: keep-alive\r\n\r\n", *target, *target)
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		res["ok"] = false
		res["error"] = err.Error()
		res["error_class"] = classify(err)
		emit(res)
		return
	}
	res["status"] = resp.StatusCode
	res["status_text"] = resp.Status
	emit(res)
}

func doAbsolute(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	rawURL := fs.String("url", "", "absolute-form URL")
	hostHeader := fs.String("host-header", "", "Host header to send")
	timeout := durationFlag(fs, "timeout", "8s", "timeout")
	extra := stringList{}
	fs.Var(&extra, "header", "extra header (repeatable)")
	_ = fs.Parse(args)

	emitUp(action)
	res := result{"action": action, "ok": true, "url": *rawURL}
	proxy, rawProxy, err := proxyAddress()
	if err != nil {
		res["ok"] = false
		res["error_class"] = "harness_error"
		res["error"] = err.Error()
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
	host := *hostHeader
	if host == "" {
		if u, err := url.Parse(*rawURL); err == nil {
			host = u.Host
		}
	}
	fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n", *rawURL, host)
	for _, h := range extra {
		fmt.Fprintf(conn, "%s\r\n", h)
	}
	fmt.Fprint(conn, "\r\n")
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodGet})
	if err != nil {
		res["ok"] = false
		res["error"] = err.Error()
		res["error_class"] = classify(err)
		emit(res)
		return
	}
	res["status"] = resp.StatusCode
	res["status_text"] = resp.Status
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	res["body_prefix"] = truncate(string(body), 256)
	emit(res)
}

func doSOCKS(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	target := fs.String("target", "", "target host:port")
	timeout := durationFlag(fs, "timeout", "5s", "timeout")
	_ = fs.Parse(args)

	emitUp(action)
	res := result{"action": action, "ok": true, "target": *target}
	proxy, rawProxy, err := proxyAddress()
	if err != nil {
		res["ok"] = false
		res["error_class"] = "harness_error"
		res["error"] = err.Error()
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
	// SOCKS5 greeting: version 5, one method, no auth.
	_, _ = conn.Write([]byte{0x05, 0x01, 0x00})
	buf := make([]byte, 2)
	n, rerr := conn.Read(buf)
	res["greeting_bytes"] = n
	res["greeting_hex"] = fmt.Sprintf("%x", buf[:n])
	if rerr != nil {
		res["ok"] = false
		res["error"] = rerr.Error()
		res["error_class"] = classify(rerr)
		emit(res)
		return
	}
	res["socks_supported"] = n == 2 && buf[0] == 0x05 && buf[1] == 0x00
	emit(res)
}

func doNumeric(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	ip := fs.String("ip", "192.0.2.66", "IPv4 address to encode")
	port := fs.Int("port", 80, "port")
	path := fs.String("path", "/", "path")
	timeout := durationFlag(fs, "timeout", "8s", "timeout")
	ca := fs.String("ca", "", "PEM CA bundle")
	_ = fs.Parse(args)

	emitUp(action)
	u, err := net.ResolveIPAddr("ip4", *ip)
	if err != nil {
		emit(result{"action": action, "ok": false, "error_class": "harness_error", "error": err.Error()})
		return
	}
	v4 := u.IP.To4()
	if v4 == nil {
		emit(result{"action": action, "ok": false, "error_class": "harness_error", "error": "not ipv4"})
		return
	}
	integer := uint32(v4[0])<<24 | uint32(v4[1])<<16 | uint32(v4[2])<<8 | uint32(v4[3])
	forms := []struct{ name, host string }{
		{"canonical", *ip},
		{"integer", strconv.FormatUint(uint64(integer), 10)},
		{"hex", fmt.Sprintf("0x%08x", integer)},
		{"octal", fmt.Sprintf("0%o", integer)},
		{"ipv4-mapped", "[::ffff:" + *ip + "]"},
		{"trailing-dot", *ip + "."},
	}
	for _, f := range forms {
		rawURL := fmt.Sprintf("http://%s:%d%s", f.host, *port, *path)
		client, err := newHTTPClient(*ca, "", "", "", false, false, false, false)
		if err != nil {
			emit(result{"action": action, "form": f.name, "url": rawURL, "ok": false, "error_class": "harness_error", "error": err.Error()})
			continue
		}
		client.Timeout = *timeout
		resp, err := client.Get(rawURL)
		res := result{"action": action, "form": f.name, "url": rawURL, "ok": err == nil}
		if err != nil {
			res["error"] = err.Error()
			res["error_class"] = classify(err)
		} else {
			res["status"] = resp.StatusCode
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
		}
		client.CloseIdleConnections()
		emit(res)
	}
}

func doHostVariants(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	host := fs.String("host", "", "allowed hostname")
	port := fs.Int("port", 80, "port")
	path := fs.String("path", "/", "path")
	timeout := durationFlag(fs, "timeout", "8s", "timeout")
	ca := fs.String("ca", "", "PEM CA bundle")
	_ = fs.Parse(args)

	emitUp(action)
	forms := []struct{ name, host string }{
		{"canonical", *host},
		{"uppercase", strings.ToUpper(*host)},
		{"trailing-dot", *host + "."},
		{"mixed-case", mixedCase(*host)},
	}
	for _, f := range forms {
		rawURL := fmt.Sprintf("http://%s:%d%s", f.host, *port, *path)
		client, err := newHTTPClient(*ca, "", "", "", false, false, false, false)
		if err != nil {
			emit(result{"action": action, "form": f.name, "url": rawURL, "ok": false, "error_class": "harness_error", "error": err.Error()})
			continue
		}
		client.Timeout = *timeout
		resp, err := client.Get(rawURL)
		res := result{"action": action, "form": f.name, "url": rawURL, "ok": err == nil}
		if err != nil {
			res["error"] = err.Error()
			res["error_class"] = classify(err)
		} else {
			res["status"] = resp.StatusCode
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
		}
		client.CloseIdleConnections()
		emit(res)
	}
}

func mixedCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i%2 == 0 {
			b.WriteString(strings.ToUpper(string(r)))
		} else {
			b.WriteString(strings.ToLower(string(r)))
		}
	}
	return b.String()
}

func doUserinfo(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	rawURL := fs.String("url", "", "URL with userinfo")
	timeout := durationFlag(fs, "timeout", "8s", "timeout")
	_ = fs.Parse(args)

	emitUp(action)
	client, err := newHTTPClient("", "", "", "", false, false, false, false)
	if err != nil {
		emit(result{"action": action, "ok": false, "error_class": "harness_error", "error": err.Error()})
		return
	}
	client.Timeout = *timeout
	resp, err := client.Get(*rawURL)
	res := result{"action": action, "url": *rawURL, "ok": err == nil}
	if err != nil {
		res["error"] = err.Error()
		res["error_class"] = classify(err)
	} else {
		res["status"] = resp.StatusCode
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
	}
	client.CloseIdleConnections()
	emit(res)
}

func doEnv(action string, _ []string) {
	emitUp(action)
	names := []string{
		"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY",
		"http_proxy", "https_proxy", "all_proxy", "no_proxy",
		"PATH",
	}
	env := map[string]string{}
	for _, n := range names {
		env[n] = os.Getenv(n)
	}
	emit(result{"action": action, "ok": true, "env": env, "ppid": os.Getppid(), "pid": os.Getpid()})
}

func doNetinfo(action string, _ []string) {
	emitUp(action)
	ifaces := []map[string]any{}
	list, err := net.Interfaces()
	if err == nil {
		for _, iface := range list {
			addrs, _ := iface.Addrs()
			as := []string{}
			for _, a := range addrs {
				as = append(as, a.String())
			}
			ifaces = append(ifaces, map[string]any{
				"name":  iface.Name,
				"flags": iface.Flags.String(),
				"addrs": as,
			})
		}
	}
	routes := ""
	if b, err := os.ReadFile("/proc/net/route"); err == nil {
		routes = string(b)
	}
	emit(result{"action": action, "ok": true, "interfaces": ifaces, "routes": routes})
}

func doFlood(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	rawURL := fs.String("url", "", "request URL")
	n := fs.Int("n", 25, "number of requests")
	conc := fs.Int("concurrency", 10, "concurrency")
	closeConn := fs.Bool("close", false, "disable keep-alives (fresh TCP connection per request)")
	proxyFromAll := fs.Bool("proxy-from-all", false, "use ALL_PROXY (curl-style) instead of Go's env lookup")
	timeout := durationFlag(fs, "timeout", "8s", "per-request timeout")
	_ = fs.Parse(args)

	emitUp(action)
	if *rawURL == "" {
		emit(result{"action": action, "ok": false, "error_class": "harness_error", "error": "missing --url"})
		return
	}
	type one struct {
		status int
		class  string
	}
	results := make(chan one, *n)
	var wg sync.WaitGroup
	sem := make(chan struct{}, *conc)
	for i := 0; i < *n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			client, err := newHTTPClient("", "", "", "", false, false, false, *proxyFromAll)
			if err != nil {
				results <- one{class: "harness_error"}
				return
			}
			client.Timeout = *timeout
			if *closeConn {
				if tr, ok := client.Transport.(*http.Transport); ok {
					tr.DisableKeepAlives = true
				}
			}
			resp, err := client.Get(*rawURL)
			if err != nil {
				results <- one{class: classify(err)}
				return
			}
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			results <- one{status: resp.StatusCode}
		}()
	}
	wg.Wait()
	close(results)
	counts := map[string]int{}
	for r := range results {
		if r.class != "" {
			counts[r.class]++
		} else {
			counts[fmt.Sprintf("status_%d", r.status)]++
		}
	}
	emit(result{"action": action, "ok": true, "requests": *n, "counts": counts})
}

func doHold(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	addr := fs.String("addr", "", "tcp address to hold open")
	seconds := fs.Int("seconds", 5, "seconds to hold")
	timeout := durationFlag(fs, "timeout", "5s", "dial timeout")
	_ = fs.Parse(args)

	emitUp(action)
	d := net.Dialer{Timeout: *timeout}
	conn, err := d.Dial("tcp", *addr)
	if err != nil {
		emit(result{"action": action, "ok": false, "error": err.Error(), "error_class": classify(err)})
		return
	}
	defer conn.Close()
	emit(result{"action": action, "ok": true, "local": conn.LocalAddr().String(), "remote": conn.RemoteAddr().String(), "held_seconds": *seconds})
	time.Sleep(time.Duration(*seconds) * time.Second)
	emit(result{"action": action, "ok": true, "held": true, "seconds": *seconds})
}

func doUnix(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	path := fs.String("path", "", "unix socket path")
	httpHost := fs.String("http-host", "", "if set, speak HTTP through the socket")
	timeout := durationFlag(fs, "timeout", "5s", "timeout")
	_ = fs.Parse(args)

	emitUp(action)
	res := result{"action": action, "ok": true, "path": *path}
	if *path == "" {
		res["ok"] = false
		res["error_class"] = "harness_error"
		res["error"] = "missing --path"
		emit(res)
		return
	}
	d := net.Dialer{Timeout: *timeout}
	conn, err := d.Dial("unix", *path)
	if err != nil {
		res["ok"] = false
		res["error"] = err.Error()
		res["error_class"] = classify(err)
		emit(res)
		return
	}
	defer conn.Close()
	res["connected"] = true
	if *httpHost != "" {
		_ = conn.SetDeadline(time.Now().Add(*timeout))
		fmt.Fprintf(conn, "GET http://%s/ HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", *httpHost, *httpHost)
		br := bufio.NewReader(conn)
		resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodGet})
		if err != nil {
			res["ok"] = false
			res["error"] = err.Error()
			res["error_class"] = classify(err)
			emit(res)
			return
		}
		res["status"] = resp.StatusCode
	}
	emit(res)
}

func doTamper(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	policy := fs.String("policy", "", "policy path")
	auditPath := fs.String("audit", "", "audit path")
	out := fs.String("out", "", "writable sandbox output path")
	extraRead := stringList{}
	extraWrite := stringList{}
	fs.Var(&extraRead, "extra-read", "extra path to read (repeatable)")
	fs.Var(&extraWrite, "extra-write", "extra path to write (repeatable)")
	_ = fs.Parse(args)

	emitUp(action)
	attempts := []result{}
	add := func(name string, err error) {
		r := result{"attempt": name, "ok": err == nil}
		if err != nil {
			r["error"] = err.Error()
			r["error_class"] = classify(err)
		}
		attempts = append(attempts, r)
	}
	for _, p := range extraRead {
		_, err := os.ReadFile(p)
		add("read:"+p, err)
	}
	for _, p := range extraWrite {
		add("write:"+p, os.WriteFile(p, []byte("warden-egress-tamper\n"), 0o600))
	}
	if *out != "" {
		control := *out + "/control.txt"
		add("control_write", os.WriteFile(control, []byte("sandbox-write-control"), 0o600))
	}
	if *policy != "" {
		_, err := os.ReadFile(*policy)
		add("policy_read", err)
		add("policy_write", os.WriteFile(*policy, []byte("network:\n  allow: ['*']\n"), 0o600))
		tmp := *policy + ".tmp"
		if err := os.WriteFile(tmp, []byte("network:\n  allow: ['*']\n"), 0o600); err != nil {
			add("policy_replace_temp", err)
		} else if err := os.Rename(tmp, *policy); err != nil {
			add("policy_replace", err)
		} else {
			add("policy_replace", nil)
		}
	}
	if *auditPath != "" {
		_, err := os.ReadFile(*auditPath)
		add("audit_read", err)
		add("audit_write", os.WriteFile(*auditPath, []byte("{}\n"), 0o600))
	}
	emit(result{"action": action, "ok": true, "attempts": attempts})
}

func doMCPStdio(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	ca := fs.String("ca", "", "PEM CA bundle")
	proxyFromAll := fs.Bool("proxy-from-all", false, "use ALL_PROXY (curl-style) instead of Go's env lookup")
	timeout := durationFlag(fs, "timeout", "15s", "request timeout")
	_ = fs.Parse(args)

	// Keep stdout clean for JSON-RPC; probe metadata goes to stderr.
	fmt.Fprintln(os.Stderr, `{"probe":"egress","action":"mcp-stdio","marker":"PROBE_UP","ok":true}`)
	client, err := newHTTPClient(*ca, "", "", "", false, false, false, *proxyFromAll)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"probe":"egress","action":"mcp-stdio","ok":false,"error":%q}`+"\n", err.Error())
		return
	}
	client.Timeout = *timeout
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		id, hasID := msg["id"]
		if !hasID {
			continue
		}
		method, _ := msg["method"].(string)
		resp := map[string]any{"jsonrpc": "2.0", "id": id}
		switch method {
		case "initialize":
			resp["result"] = map[string]any{
				"protocolVersion": "2025-11-25",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "warden-egress-probe", "version": "1"},
			}
		case "tools/list":
			resp["result"] = map[string]any{"tools": []map[string]any{{
				"name":        "egress",
				"description": "fetch a URL through the sandbox network policy",
				"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"url": map[string]any{"type": "string"}}, "required": []string{"url"}},
			}}}
		case "tools/call":
			params, _ := msg["params"].(map[string]any)
			argsMap, _ := params["arguments"].(map[string]any)
			rawURL, _ := argsMap["url"].(string)
			out := map[string]any{"url": rawURL}
			meta := map[string]any{"probe": "egress", "action": "mcp-call", "url": rawURL, "ok": false}
			if rawURL == "" {
				out["ok"] = false
				meta["error_class"] = "missing_url"
			} else if resp2, err := client.Get(rawURL); err != nil {
				out["ok"] = false
				out["error"] = err.Error()
				out["error_class"] = classify(err)
				meta["error"] = err.Error()
				meta["error_class"] = classify(err)
			} else {
				out["ok"] = true
				out["status"] = resp2.StatusCode
				meta["ok"] = err == nil
				meta["status"] = resp2.StatusCode
				_, _ = io.Copy(io.Discard, io.LimitReader(resp2.Body, 4096))
				resp2.Body.Close()
			}
			b, _ := json.Marshal(meta)
			fmt.Fprintln(os.Stderr, string(b))
			resp["result"] = map[string]any{"content": []map[string]any{{"type": "text", "text": fmt.Sprintf("%v", out)}}, "isError": false}
		default:
			resp["error"] = map[string]any{"code": -32601, "message": "method not found"}
		}
		b, _ := json.Marshal(resp)
		fmt.Println(string(b))
	}
	client.CloseIdleConnections()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: probe <action> [flags]")
		os.Exit(2)
	}
	action := os.Args[1]
	args := os.Args[2:]
	switch action {
	case "http", "https":
		doHTTP(action, args)
	case "raw":
		doRaw(action, args)
	case "dns":
		doDNS(action, args)
	case "connect":
		doConnect(action, args)
	case "absolute":
		doAbsolute(action, args)
	case "socks":
		doSOCKS(action, args)
	case "numeric":
		doNumeric(action, args)
	case "host-variants":
		doHostVariants(action, args)
	case "userinfo":
		doUserinfo(action, args)
	case "env":
		doEnv(action, args)
	case "netinfo":
		doNetinfo(action, args)
	case "flood":
		doFlood(action, args)
	case "hold":
		doHold(action, args)
	case "unix":
		doUnix(action, args)
	case "tamper":
		doTamper(action, args)
	case "mcp-stdio":
		doMCPStdio(action, args)
	case "exec":
		doExec(action, args)
	case "exec-self":
		doExecSelf(action, args)
	case "proxy-serve":
		doProxyServe(action, args)
	case "child-proxy":
		doChildProxy(action, args)
	case "env-tamper":
		doEnvTamper(action, args)
	case "ws":
		doWS(action, args)
	case "unshare":
		doUnshare(action, args)
	case "kill-parent":
		doKillParent(action, args)
	case "signal-parent":
		doSignalParent(action, args)
	default:
		fmt.Fprintf(os.Stderr, "unknown action %q\n", action)
		os.Exit(2)
	}
}
