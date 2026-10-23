package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Hit is one observation at a local collector.
type Hit struct {
	Collector     string              `json:"collector"`
	Proto         string              `json:"proto"`
	Time          time.Time           `json:"time"`
	Src           string              `json:"src,omitempty"`
	Dst           string              `json:"dst,omitempty"`
	SNI           string              `json:"sni,omitempty"`
	Host          string              `json:"host,omitempty"`
	Method        string              `json:"method,omitempty"`
	Path          string              `json:"path,omitempty"`
	Query         string              `json:"query,omitempty"`
	Headers       map[string][]string `json:"headers,omitempty"`
	PayloadPrefix string              `json:"payload_prefix,omitempty"`
	Token         string              `json:"token,omitempty"`
	TLSVersion    uint16              `json:"tls_version,omitempty"`
	ProtoVersion  string              `json:"proto_version,omitempty"`
	FakeProxy     bool                `json:"fake_proxy,omitempty"`
	Upstream      string              `json:"upstream,omitempty"`
	UpstreamOK    bool                `json:"upstream_ok,omitempty"`
}

// DNSHit is one DNS query observed by the test resolver.
type DNSHit struct {
	Time      time.Time `json:"time"`
	Name      string    `json:"name"`
	QType     string    `json:"qtype"`
	Transport string    `json:"transport"`
	Src       string    `json:"src,omitempty"`
	Answer    []string  `json:"answer,omitempty"`
	RCode     int       `json:"rcode"`
}

type dnsEntry struct {
	v4     net.IP
	v6     net.IP
	flipV4 []net.IP
	rcode  int
}

type collectorConfig struct {
	Mode       string // "netns" or "host"
	Zone       string
	DNSIP      string
	AllowedIP  string
	AllowedIP6 string
	DeniedIP   string
	DeniedIP6  string
	MetaIP     string
	PrivateIP  string
	LoopIP     string
	LANIP      string
	Cert       tls.Certificate
}

// Collectors owns every local test endpoint plus the DNS resolver.
type Collectors struct {
	cfg collectorConfig

	mu         sync.Mutex
	hits       []Hit
	dns        []DNSHit
	closers    []io.Closer
	dnsClosers []io.Closer
	ports      map[string]int
	addrs      map[string]string
	records    map[string]dnsEntry
	flips      map[string]*uint32
	hitsFile   *os.File
	dnsFile    *os.File
	logDir     string
	closed     bool
	v6OK       bool

	httpDefs      map[string]httpDef
	httpServers   map[string]*http.Server
	httpListeners map[string]net.Listener
}

type httpDef struct {
	ip      string
	withTLS bool
}

func newCollectors(cfg collectorConfig, logDir string) *Collectors {
	return &Collectors{
		cfg:           cfg,
		ports:         map[string]int{},
		addrs:         map[string]string{},
		records:       map[string]dnsEntry{},
		flips:         map[string]*uint32{},
		logDir:        logDir,
		httpDefs:      map[string]httpDef{},
		httpServers:   map[string]*http.Server{},
		httpListeners: map[string]net.Listener{},
	}
}

func (c *Collectors) record(h Hit) {
	h.Time = time.Now().UTC()
	c.mu.Lock()
	c.hits = append(c.hits, h)
	if c.hitsFile != nil {
		b, _ := json.Marshal(h)
		_, _ = c.hitsFile.Write(append(b, '\n'))
	}
	c.mu.Unlock()
}

func (c *Collectors) recordDNS(h DNSHit) {
	h.Time = time.Now().UTC()
	c.mu.Lock()
	c.dns = append(c.dns, h)
	if c.dnsFile != nil {
		b, _ := json.Marshal(h)
		_, _ = c.dnsFile.Write(append(b, '\n'))
	}
	c.mu.Unlock()
}

func (c *Collectors) addCloser(cl io.Closer) {
	c.mu.Lock()
	c.closers = append(c.closers, cl)
	c.mu.Unlock()
}

func (c *Collectors) addDNSCloser(cl io.Closer) {
	c.mu.Lock()
	c.closers = append(c.closers, cl)
	c.dnsClosers = append(c.dnsClosers, cl)
	c.mu.Unlock()
}

// StopDNS closes the test resolver. StartDNS brings it back with a fresh
// socket; used by the resolver-failure fail-closed case.
func (c *Collectors) StopDNS() {
	c.mu.Lock()
	closers := c.dnsClosers
	c.dnsClosers = nil
	c.mu.Unlock()
	for _, cl := range closers {
		_ = cl.Close()
	}
}

func (c *Collectors) StartDNS() {
	c.startDNS()
}

func (c *Collectors) setPort(name, addr string) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return
	}
	var p int
	fmt.Sscanf(port, "%d", &p)
	c.mu.Lock()
	c.ports[name] = p
	c.addrs[name] = addr
	c.mu.Unlock()
}

func (c *Collectors) Port(name string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ports[name]
}

func (c *Collectors) Addr(name string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.addrs[name]
}

func (c *Collectors) registerDNS(name string, e dnsEntry) {
	c.mu.Lock()
	c.records[strings.ToLower(strings.TrimSuffix(name, "."))] = e
	c.mu.Unlock()
}

func (c *Collectors) Hits() []Hit {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Hit, len(c.hits))
	copy(out, c.hits)
	return out
}

func (c *Collectors) DNSQueries() []DNSHit {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]DNSHit, len(c.dns))
	copy(out, c.dns)
	return out
}

// HitsForToken returns collector hits whose content references token. This
// is how the runner scopes observations to one case even though collectors
// are shared.
func (c *Collectors) HitsForToken(token string) []Hit {
	var out []Hit
	for _, h := range c.Hits() {
		if h.Token == token || strings.Contains(h.Path, token) ||
			strings.Contains(h.Query, token) || strings.Contains(h.Host, token) ||
			strings.Contains(h.PayloadPrefix, token) || strings.Contains(h.Upstream, token) ||
			strings.Contains(fmt.Sprint(h.Headers), token) {
			out = append(out, h)
		}
	}
	return out
}

func (c *Collectors) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	closers := c.closers
	c.closers = nil
	hitsFile, dnsFile := c.hitsFile, c.dnsFile
	c.hitsFile, c.dnsFile = nil, nil
	c.mu.Unlock()
	for _, cl := range closers {
		_ = cl.Close()
	}
	if hitsFile != nil {
		_ = hitsFile.Close()
	}
	if dnsFile != nil {
		_ = dnsFile.Close()
	}
}

// Start builds every listener. Workloads must be started after this.
func (c *Collectors) Start() error {
	if err := os.MkdirAll(c.logDir, 0o755); err != nil {
		return err
	}
	var err error
	c.hitsFile, err = os.OpenFile(filepath.Join(c.logDir, "collector-hits.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	c.dnsFile, err = os.OpenFile(filepath.Join(c.logDir, "dns.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	c.startDNS()

	allowedHost := c.cfg.AllowedIP
	deniedHost := c.cfg.DeniedIP

	if err := c.startHTTP("allowed-http", allowedHost, false); err != nil {
		return fmt.Errorf("allowed-http: %w", err)
	}
	// A second HTTP service on the same allowed IP but a different port. It is
	// deliberately NOT a forbidden collector: it exists to observe that host
	// grants are port-agnostic (the policy schema has no port field).
	if err := c.startHTTP("allowed-alt-http", allowedHost, false); err != nil {
		return fmt.Errorf("allowed-alt-http: %w", err)
	}
	if err := c.startHTTP("allowed-https", allowedHost, true); err != nil {
		return fmt.Errorf("allowed-https: %w", err)
	}
	if err := c.startHTTP("denied-http", deniedHost, false); err != nil {
		return fmt.Errorf("denied-http: %w", err)
	}
	if err := c.startHTTP("denied-https", deniedHost, true); err != nil {
		return fmt.Errorf("denied-https: %w", err)
	}
	if err := c.startTCPSink("allowed-tcp", allowedHost); err != nil {
		return fmt.Errorf("allowed-tcp: %w", err)
	}
	if err := c.startTCPSink("denied-tcp", deniedHost); err != nil {
		return fmt.Errorf("denied-tcp: %w", err)
	}
	if err := c.startUDPSink("denied-udp", deniedHost); err != nil {
		return fmt.Errorf("denied-udp: %w", err)
	}
	if err := c.startTCPSink("metadata-tcp", c.cfg.MetaIP); err != nil {
		return fmt.Errorf("metadata-tcp: %w", err)
	}
	if err := c.startUDPSink("metadata-udp", c.cfg.MetaIP); err != nil {
		return fmt.Errorf("metadata-udp: %w", err)
	}
	if err := c.startTCPSink("private-tcp", c.cfg.PrivateIP); err != nil {
		return fmt.Errorf("private-tcp: %w", err)
	}
	if c.cfg.Mode == "host" {
		// In host mode the workload shares the loopback stack; bind the
		// bypass collector on loopback so a Seatbelt/AppContainer miss is
		// observable.
		if err := c.startTCPSink("loopback-tcp", c.cfg.LoopIP); err != nil {
			return fmt.Errorf("loopback-tcp: %w", err)
		}
	} else {
		if err := c.startTCPSink("loopback-tcp", c.cfg.LoopIP); err != nil {
			return fmt.Errorf("loopback-tcp: %w", err)
		}
	}
	if err := c.startFakeProxy(); err != nil {
		return fmt.Errorf("fake-proxy: %w", err)
	}
	if err := c.startHTTPSpecial(); err != nil {
		return fmt.Errorf("special: %w", err)
	}
	if c.cfg.AllowedIP6 != "" && c.cfg.DeniedIP6 != "" {
		if err := c.startIPv6(); err != nil {
			// IPv6 unavailable is a capability limitation, not a harness
			// failure; the runner marks IPv6 cases UNSUPPORTED.
			c.mu.Lock()
			c.v6OK = false
			c.mu.Unlock()
		} else {
			c.mu.Lock()
			c.v6OK = true
			c.mu.Unlock()
		}
	}
	return nil
}

func (c *Collectors) V6OK() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.v6OK
}

func (c *Collectors) listen(network, addr string) (net.Listener, error) {
	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	c.addCloser(ln)
	return ln, nil
}

func (c *Collectors) startHTTP(name, ip string, withTLS bool) error {
	c.mu.Lock()
	c.httpDefs[name] = httpDef{ip: ip, withTLS: withTLS}
	c.mu.Unlock()
	return c.startHTTPNow(name, ip, withTLS)
}

// StopHTTP closes a managed HTTP collector. StartHTTP recreates it, which
// is how the collector-failure case proves fail-closed behavior.
func (c *Collectors) StopHTTP(name string) error {
	c.mu.Lock()
	srv := c.httpServers[name]
	ln := c.httpListeners[name]
	delete(c.httpServers, name)
	delete(c.httpListeners, name)
	c.mu.Unlock()
	if srv != nil {
		_ = srv.Close()
	}
	if ln != nil {
		_ = ln.Close()
	}
	return nil
}

func (c *Collectors) StartHTTP(name string) error {
	c.mu.Lock()
	def, ok := c.httpDefs[name]
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("collector %s was never defined", name)
	}
	return c.startHTTPNow(name, def.ip, def.withTLS)
}

func (c *Collectors) startHTTPNow(name, ip string, withTLS bool) error {
	addr := net.JoinHostPort(ip, "0")
	ln, err := c.listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Handler:           c.httpHandler(name, withTLS),
		ReadHeaderTimeout: 10 * time.Second,
	}
	if withTLS {
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{c.cfg.Cert},
			MinVersion:   tls.VersionTLS10,
		}
		// Record TLS handshakes (including SNI) even when the client later
		// fails verification: the attempt itself is evidence.
		tlsCfg.GetConfigForClient = func(hi *tls.ClientHelloInfo) (*tls.Config, error) {
			c.record(Hit{Collector: name, Proto: "tls-hello", SNI: hi.ServerName, Src: remoteOf(hi.Conn), Dst: localOf(hi.Conn)})
			return nil, nil
		}
		srv.TLSConfig = tlsCfg
	}
	c.setPort(name, ln.Addr().String())
	c.mu.Lock()
	c.httpServers[name] = srv
	c.httpListeners[name] = ln
	c.mu.Unlock()
	go func() {
		if withTLS {
			// ServeTLS (rather than Serve on a tls.Listener) enables ALPN and
			// HTTP/2 so the h2 case exercises the real transport.
			_ = srv.ServeTLS(ln, "", "")
			return
		}
		_ = srv.Serve(ln)
	}()
	c.addCloser(srv)
	return nil
}

func remoteOf(conn net.Conn) string {
	if conn == nil || conn.RemoteAddr() == nil {
		return ""
	}
	return conn.RemoteAddr().String()
}

func localOf(conn net.Conn) string {
	if conn == nil || conn.LocalAddr() == nil {
		return ""
	}
	return conn.LocalAddr().String()
}

func (c *Collectors) httpHandler(name string, withTLS bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		h := Hit{
			Collector:     name,
			Proto:         "http",
			Host:          r.Host,
			Method:        r.Method,
			Path:          r.URL.Path,
			Query:         r.URL.RawQuery,
			Headers:       r.Header,
			PayloadPrefix: truncate(string(body), 256),
			Src:           r.RemoteAddr,
			ProtoVersion:  r.Proto,
			Token:         extractToken(r, body),
		}
		if withTLS && r.TLS != nil {
			h.Proto = "https"
			h.SNI = r.TLS.ServerName
			h.TLSVersion = r.TLS.Version
		}
		c.record(h)

		switch {
		case strings.HasPrefix(r.URL.Path, "/redirect"):
			to := r.URL.Query().Get("to")
			if to == "" {
				to = "http://127.0.0.1:1/redirect-missing-target"
			}
			if r.URL.Path == "/redirect-chain" {
				// Two-hop chain: this hop 302s to the single-hop redirect
				// endpoint, which then 302s to the final (denied) target.
				to = "/redirect-denied?to=" + url.QueryEscape(to)
			}
			w.Header().Set("Location", to)
			w.WriteHeader(http.StatusFound)
			return
		case r.URL.Path == "/slow":
			// Parse into an int: scanning %d directly into a time.Duration
			// yields nanoseconds, not seconds.
			secs := 2
			fmt.Sscanf(r.URL.Query().Get("delay"), "%d", &secs)
			if secs < 0 {
				secs = 0
			}
			if secs > 10 {
				secs = 10
			}
			time.Sleep(time.Duration(secs) * time.Second)
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "slow-ok")
			return
		case r.URL.Path == "/large":
			n := 1024 * 1024
			fmt.Sscanf(r.URL.Query().Get("bytes"), "%d", &n)
			if n < 0 || n > 8*1024*1024 {
				n = 8 * 1024 * 1024
			}
			w.WriteHeader(http.StatusOK)
			chunk := make([]byte, 64*1024)
			for i := range chunk {
				chunk[i] = 'A'
			}
			for written := 0; written < n; {
				m := len(chunk)
				if n-written < m {
					m = n - written
				}
				k, err := w.Write(chunk[:m])
				if err != nil {
					return
				}
				written += k
			}
			return
		case r.URL.Path == "/sse":
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			fl, _ := w.(http.Flusher)
			for i := 0; i < 3; i++ {
				fmt.Fprintf(w, "data: event-%d-%s\n\n", i, extractToken(r, nil))
				if fl != nil {
					fl.Flush()
				}
				time.Sleep(100 * time.Millisecond)
			}
			return
		case r.URL.Path == "/ws":
			key := r.Header.Get("Sec-WebSocket-Key")
			if key == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			accept := websocketAccept(key)
			w.Header().Set("Upgrade", "websocket")
			w.Header().Set("Connection", "Upgrade")
			w.Header().Set("Sec-WebSocket-Accept", accept)
			hj, ok := w.(http.Hijacker)
			if !ok {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			conn, buf, err := hj.Hijack()
			if err != nil {
				return
			}
			defer conn.Close()
			_, _ = buf.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\n\r\n")
			_ = buf.Flush()
			// One unmasked text frame from server to client.
			payload := []byte("hello-from-collector")
			frame := []byte{0x81, byte(len(payload))}
			frame = append(frame, payload...)
			_, _ = conn.Write(frame)
			return
		case r.URL.Path == "/echo":
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"method":      r.Method,
				"host":        r.Host,
				"url":         r.URL.String(),
				"headers":     r.Header,
				"body":        truncate(string(body), 512),
				"body_len":    len(body),
				"body_capped": len(body) == 64*1024,
				"token":       extractToken(r, body),
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		case r.URL.Path == "/hold":
			seconds := 30
			fmt.Sscanf(r.URL.Query().Get("seconds"), "%d", &seconds)
			if seconds < 1 || seconds > 120 {
				seconds = 30
			}
			fl, _ := w.(http.Flusher)
			w.WriteHeader(http.StatusOK)
			if fl != nil {
				fl.Flush()
			}
			for i := 0; i < seconds*2; i++ {
				select {
				case <-r.Context().Done():
					return
				case <-time.After(500 * time.Millisecond):
					_, _ = io.WriteString(w, " ")
					if fl != nil {
						fl.Flush()
					}
				}
			}
			return
		default:
			w.Header().Set("X-Egress-Token", extractToken(r, body))
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "collector=%s token=%s host=%s path=%s\n", name, extractToken(r, body), r.Host, r.URL.Path)
			return
		}
	})
}

func websocketAccept(key string) string {
	h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h[:])
}

func extractToken(r *http.Request, body []byte) string {
	if v := r.URL.Query().Get("token"); v != "" {
		return v
	}
	if v := r.Header.Get("X-Egress-Token"); v != "" {
		return v
	}
	// Path convention: /<token> or /token/<token>
	for _, part := range strings.Split(r.URL.Path, "/") {
		if strings.HasPrefix(part, "tok-") {
			return part
		}
	}
	if len(body) > 0 && strings.HasPrefix(string(body), "tok-") {
		return strings.Fields(string(body))[0]
	}
	return ""
}

func (c *Collectors) startTCPSink(name, ip string) error {
	addr := net.JoinHostPort(ip, "0")
	ln, err := c.listen("tcp", addr)
	if err != nil {
		return err
	}
	c.setPort(name, ln.Addr().String())
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_ = conn.SetReadDeadline(time.Now().Add(1200 * time.Millisecond))
				buf := make([]byte, 4096)
				n, _ := conn.Read(buf)
				token := ""
				if n > 0 {
					token = firstToken(buf[:n])
				}
				c.record(Hit{
					Collector: name, Proto: "tcp",
					Src: remoteOf(conn), Dst: localOf(conn),
					PayloadPrefix: truncate(string(buf[:n]), 256), Token: token,
				})
			}(conn)
		}
	}()
	return nil
}

func (c *Collectors) startUDPSink(name, ip string) error {
	addr := net.JoinHostPort(ip, "0")
	laddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}
	c.setPort(name, conn.LocalAddr().String())
	c.addCloser(conn)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, src, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			c.record(Hit{
				Collector: name, Proto: "udp",
				Src: src.String(), Dst: conn.LocalAddr().String(),
				PayloadPrefix: truncate(string(buf[:n]), 256), Token: firstToken(buf[:n]),
			})
		}
	}()
	return nil
}

func firstToken(b []byte) string {
	s := strings.TrimSpace(string(b))
	for _, part := range strings.Fields(s) {
		if strings.HasPrefix(part, "tok-") {
			return part
		}
	}
	return ""
}

// startIPv6 starts the IPv6 collectors. Any bind failure means the host or
// sandbox lacks usable IPv6; callers turn that into UNSUPPORTED.
func (c *Collectors) startIPv6() error {
	if err := c.startHTTP("v6-allowed-http", c.cfg.AllowedIP6, false); err != nil {
		return err
	}
	if err := c.startHTTP("v6-allowed-https", c.cfg.AllowedIP6, true); err != nil {
		return err
	}
	if err := c.startTCPSink("v6-denied-tcp", c.cfg.DeniedIP6); err != nil {
		return err
	}
	if err := c.startUDPSink("v6-denied-udp", c.cfg.DeniedIP6); err != nil {
		return err
	}
	return nil
}

// startHTTPSpecial starts the DoT/DoH endpoints used by DNS tests.
func (c *Collectors) startHTTPSpecial() error {
	// DoT on the allowed IP, port 853 (root inside the test netns).
	dotAddr := net.JoinHostPort(c.cfg.AllowedIP, "853")
	ln, err := net.Listen("tcp", dotAddr)
	if err == nil {
		tlsCfg := &tls.Config{Certificates: []tls.Certificate{c.cfg.Cert}, MinVersion: tls.VersionTLS10}
		tlsLn := tls.NewListener(ln, tlsCfg)
		c.setPort("dot", tlsLn.Addr().String())
		c.addCloser(tlsLn)
		go func() {
			for {
				conn, err := tlsLn.Accept()
				if err != nil {
					return
				}
				go func(conn net.Conn) {
					defer conn.Close()
					_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
					buf := make([]byte, 4096)
					n, _ := conn.Read(buf)
					sni := ""
					if tc, ok := conn.(*tls.Conn); ok {
						_ = tc.Handshake()
						sni = tc.ConnectionState().ServerName
					}
					c.record(Hit{Collector: "dot", Proto: "dot", SNI: sni,
						Src: remoteOf(conn), Dst: localOf(conn),
						PayloadPrefix: truncate(string(buf[:n]), 256), Token: firstToken(buf[:n])})
				}(conn)
			}
		}()
	}
	// DoH on the allowed IP, port 443.
	dohAddr := net.JoinHostPort(c.cfg.AllowedIP, "443")
	ln2, err := net.Listen("tcp", dohAddr)
	if err == nil {
		tlsCfg := &tls.Config{Certificates: []tls.Certificate{c.cfg.Cert}, MinVersion: tls.VersionTLS10}
		tlsLn := tls.NewListener(ln2, tlsCfg)
		c.setPort("doh", tlsLn.Addr().String())
		c.addCloser(tlsLn)
		srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(io.LimitReader(r.Body, 64*1024))
			c.record(Hit{Collector: "doh", Proto: "doh", Host: r.Host, Path: r.URL.Path,
				Query: r.URL.RawQuery, Src: r.RemoteAddr, PayloadPrefix: truncate(string(body), 256),
				Token: extractToken(r, body), SNI: tlsServerName(r)})
			w.Header().Set("Content-Type", "application/dns-message")
			w.WriteHeader(http.StatusOK)
		}), ReadHeaderTimeout: 10 * time.Second}
		go func() { _ = srv.Serve(tlsLn) }()
		c.addCloser(srv)
	}
	return nil
}

func tlsServerName(r *http.Request) string {
	if r.TLS != nil {
		return r.TLS.ServerName
	}
	return ""
}

// startFakeProxy starts a permissive HTTP proxy on host loopback. A sandbox
// that can reach it and make it dial a denied destination has a real bypass
// path outside Warden's control.
func (c *Collectors) startFakeProxy() error {
	ln, err := net.Listen("tcp", net.JoinHostPort(c.cfg.LoopIP, "0"))
	if err != nil {
		return err
	}
	c.setPort("fake-proxy", ln.Addr().String())
	c.addCloser(ln)
	srv := &http.Server{Handler: http.HandlerFunc(c.fakeProxyHandler), ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	c.addCloser(srv)
	return nil
}

func (c *Collectors) fakeProxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		target := r.Host
		conn, err := net.DialTimeout("tcp", target, 3*time.Second)
		ok := err == nil
		c.record(Hit{Collector: "fake-proxy", Proto: "connect", Host: target, Upstream: target, UpstreamOK: ok, FakeProxy: true, Token: firstToken([]byte(target))})
		if err != nil {
			http.Error(w, "fake proxy upstream failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		hj, okHj := w.(http.Hijacker)
		if !okHj {
			conn.Close()
			http.Error(w, "no hijack", http.StatusInternalServerError)
			return
		}
		client, buf, err := hj.Hijack()
		if err != nil {
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
	target := r.URL.Host
	if target == "" {
		target = r.Host
	}
	u := r.URL.String()
	req, err := http.NewRequest(r.Method, u, r.Body)
	if err == nil {
		req.Host = r.Host
		out, derr := (&http.Client{Timeout: 5 * time.Second}).Do(req)
		ok := derr == nil
		c.record(Hit{Collector: "fake-proxy", Proto: "http", Host: r.Host, Path: r.URL.Path, Query: r.URL.RawQuery,
			Upstream: target, UpstreamOK: ok, FakeProxy: true, Token: extractToken(r, nil)})
		if derr == nil {
			defer out.Body.Close()
			copyHeaderLocal(w.Header(), out.Header)
			w.WriteHeader(out.StatusCode)
			_, _ = io.Copy(io.Discard, io.LimitReader(out.Body, 1<<20))
			_, _ = io.WriteString(w, "fake-proxy-forwarded\n")
			return
		}
	}
	http.Error(w, "fake proxy cannot forward", http.StatusBadGateway)
}

func copyHeaderLocal(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// StartUnixFake serves the fake-proxy protocol on a Unix socket inside a
// path granted to the sandbox. Used to prove what a granted AF_UNIX path
// exposes.
func (c *Collectors) StartUnixFake(path string) error {
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	c.addCloser(ln)
	srv := &http.Server{Handler: http.HandlerFunc(c.fakeProxyHandler), ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	c.addCloser(srv)
	return nil
}

// StartUnixEcho records AF_UNIX connections and answers every HTTP request
// with 403. It models a granted host-side control socket that is reachable
// but provides no egress on its own.
func (c *Collectors) StartUnixEcho(path string) error {
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	c.addCloser(ln)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
				buf := make([]byte, 2048)
				n, _ := conn.Read(buf)
				c.record(Hit{Collector: "unix-fake", Proto: "unix", Src: remoteOf(conn), Dst: localOf(conn),
					PayloadPrefix: truncate(string(buf[:n]), 256), Token: firstToken(buf[:n])})
				_, _ = conn.Write([]byte("HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"))
			}(conn)
		}
	}()
	return nil
}

// ---- DNS ----

func (c *Collectors) startDNS() {
	addr := net.JoinHostPort(c.cfg.DNSIP, "53")
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err == nil {
		udpConn, err := net.ListenUDP("udp", udpAddr)
		if err == nil {
			c.addDNSCloser(udpConn)
			go c.dnsUDP(udpConn)
		}
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err == nil {
		ln, err := net.ListenTCP("tcp", tcpAddr)
		if err == nil {
			c.addDNSCloser(ln)
			go c.dnsTCP(ln)
		}
	}
}

func (c *Collectors) dnsUDP(conn *net.UDPConn) {
	buf := make([]byte, 4096)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		resp := c.handleDNS(buf[:n], "udp", src.String())
		if resp != nil {
			_, _ = conn.WriteToUDP(resp, src)
		}
	}
}

func (c *Collectors) dnsTCP(ln *net.TCPListener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go func(conn net.Conn) {
			defer conn.Close()
			for {
				_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
				var lbuf [2]byte
				if _, err := io.ReadFull(conn, lbuf[:]); err != nil {
					return
				}
				l := int(binary.BigEndian.Uint16(lbuf[:]))
				if l <= 0 || l > 4096 {
					return
				}
				msg := make([]byte, l)
				if _, err := io.ReadFull(conn, msg); err != nil {
					return
				}
				resp := c.handleDNS(msg, "tcp", remoteOf(conn))
				if resp == nil {
					return
				}
				out := make([]byte, 2+len(resp))
				binary.BigEndian.PutUint16(out, uint16(len(resp)))
				copy(out[2:], resp)
				if _, err := conn.Write(out); err != nil {
					return
				}
			}
		}(conn)
	}
}

// handleDNS parses enough of a query to answer A/AAAA from the per-case
// record map and logs the query for evidence.
func (c *Collectors) handleDNS(msg []byte, transport, src string) []byte {
	if len(msg) < 12 {
		return nil
	}
	id := binary.BigEndian.Uint16(msg[0:2])
	qd := int(binary.BigEndian.Uint16(msg[4:6]))
	if qd != 1 {
		return nil
	}
	off := 12
	name, off, ok := parseQName(msg, off)
	if !ok {
		return nil
	}
	if off+4 > len(msg) {
		return nil
	}
	qtype := binary.BigEndian.Uint16(msg[off : off+2])
	questionEnd := off + 4
	qtypeName := map[uint16]string{1: "A", 28: "AAAA", 16: "TXT", 5: "CNAME", 255: "ANY"}[qtype]
	if qtypeName == "" {
		qtypeName = fmt.Sprintf("TYPE%d", qtype)
	}

	key := strings.ToLower(strings.TrimSuffix(name, "."))
	c.mu.Lock()
	entry, found := c.records[key]
	flip := c.flips[key]
	c.mu.Unlock()

	var answers [][]byte
	rcode := 0
	answerNames := []string{}
	if !found {
		rcode = 3 // NXDOMAIN
	} else if entry.rcode != 0 {
		rcode = entry.rcode
	} else {
		switch qtype {
		case 1:
			ip := entry.v4
			if len(entry.flipV4) > 0 && flip != nil {
				// Only A answers advance the flip counter; AAAA lookups must
				// not consume a rebinding step.
				c.mu.Lock()
				flipIdx := *flip
				*flip = flipIdx + 1
				c.mu.Unlock()
				ip = entry.flipV4[int(flipIdx)%len(entry.flipV4)]
			}
			if ip != nil {
				answers = append(answers, dnsAnswer(ip.To4()))
				answerNames = append(answerNames, ip.String())
			}
		case 28:
			if entry.v6 != nil {
				answers = append(answers, dnsAnswer(entry.v6.To16()))
				answerNames = append(answerNames, entry.v6.String())
			}
		case 255:
			if entry.v4 != nil {
				answers = append(answers, dnsAnswer(entry.v4.To4()))
				answerNames = append(answerNames, entry.v4.String())
			}
			if entry.v6 != nil {
				answers = append(answers, dnsAnswer(entry.v6.To16()))
				answerNames = append(answerNames, entry.v6.String())
			}
		}
	}

	c.recordDNS(DNSHit{Name: key, QType: qtypeName, Transport: transport, Src: src, Answer: answerNames, RCode: rcode})

	flags := uint16(0x8180) // QR, RD, RA
	if rcode != 0 {
		flags = 0x8180 | uint16(rcode)
	}
	resp := make([]byte, 12)
	binary.BigEndian.PutUint16(resp[0:2], id)
	binary.BigEndian.PutUint16(resp[2:4], flags)
	binary.BigEndian.PutUint16(resp[4:6], 1)
	binary.BigEndian.PutUint16(resp[6:8], uint16(len(answers)))
	resp = append(resp, msg[12:questionEnd]...)
	for _, a := range answers {
		resp = append(resp, a...)
	}
	return resp
}

func dnsAnswer(rdata []byte) []byte {
	out := make([]byte, 0, 12+len(rdata))
	out = append(out, 0xC0, 0x0C)
	qtype := uint16(1)
	if len(rdata) == 16 {
		qtype = 28
	}
	var t [2]byte
	binary.BigEndian.PutUint16(t[:], qtype)
	out = append(out, t[:]...)
	binary.BigEndian.PutUint16(t[:], 1) // IN
	out = append(out, t[:]...)
	var ttl [4]byte
	binary.BigEndian.PutUint32(ttl[:], 1)
	out = append(out, ttl[:]...)
	binary.BigEndian.PutUint16(t[:], uint16(len(rdata)))
	out = append(out, t[:]...)
	out = append(out, rdata...)
	return out
}

// parseQName decodes a DNS name, following compression pointers within the
// message. Returns the name and the offset just past the question name.
func parseQName(msg []byte, off int) (string, int, bool) {
	var labels []string
	pos := off
	jumped := false
	next := off
	for steps := 0; steps < 128; steps++ {
		if pos >= len(msg) {
			return "", 0, false
		}
		l := int(msg[pos])
		if l == 0 {
			if !jumped {
				next = pos + 1
			}
			return strings.Join(labels, "."), next, true
		}
		if l&0xC0 == 0xC0 {
			if pos+1 >= len(msg) {
				return "", 0, false
			}
			ptr := int(binary.BigEndian.Uint16(msg[pos:pos+2]) & 0x3FFF)
			if !jumped {
				next = pos + 2
				jumped = true
			}
			pos = ptr
			continue
		}
		if pos+1+l > len(msg) {
			return "", 0, false
		}
		labels = append(labels, string(msg[pos+1:pos+1+l]))
		pos += 1 + l
	}
	return "", 0, false
}

// RegisterCaseDNS installs the per-case names. Call before each case.
func (c *Collectors) RegisterCaseDNS(caseID string, vars map[string]string, allowV6 bool) {
	zone := c.cfg.Zone
	a := c.cfg.AllowedIP
	d := c.cfg.DeniedIP
	set := func(name string, e dnsEntry) {
		c.registerDNS(name, e)
		c.mu.Lock()
		cnt := uint32(0)
		c.flips[strings.ToLower(name)] = &cnt
		c.mu.Unlock()
	}
	mk := func(v4, v6 string) dnsEntry {
		e := dnsEntry{}
		if v4 != "" {
			e.v4 = net.ParseIP(v4)
		}
		if v6 != "" {
			e.v6 = net.ParseIP(v6)
		}
		return e
	}
	set(caseID+"."+zone, mk(a, ""))
	set("denied-"+caseID+"."+zone, mk(d, ""))
	set("leak-"+caseID+"."+zone, mk(a, ""))
	rebind := dnsEntry{v4: net.ParseIP(c.cfg.LoopIP), flipV4: []net.IP{net.ParseIP(a), net.ParseIP(c.cfg.LoopIP)}}
	set("rebind-"+caseID+"."+zone, rebind)
	set("loop-"+caseID+"."+zone, mk(c.cfg.LoopIP, ""))
	set("meta-"+caseID+"."+zone, mk(c.cfg.MetaIP, ""))
	set("v4-"+caseID+"."+zone, mk(a, ""))
	if allowV6 {
		set("v6-"+caseID+"."+zone, mk("", c.cfg.AllowedIP6))
	}
	set("wild-"+caseID+"."+zone, mk(a, ""))
	set("dot-"+caseID+"."+zone, mk(a, ""))
	set("doh-"+caseID+"."+zone, mk(a, ""))
	set(zone, mk(d, ""))
	set(zone+".evil.test", mk(d, ""))
	_ = vars
}

// ---- TLS material ----

// generateTLSMaterial creates a throwaway CA and a leaf covering the run's
// test zone. It never touches the host trust store.
func generateTLSMaterial(zone string) (caPEM []byte, cert tls.Certificate, err error) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, tls.Certificate{}, err
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Warden Egress Test CA"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, tls.Certificate{}, err
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return nil, tls.Certificate{}, err
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, tls.Certificate{}, err
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: zone},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{zone, "*." + zone},
		IPAddresses: []net.IP{
			net.ParseIP("192.0.2.1"), net.ParseIP("192.0.2.66"),
			net.ParseIP("2001:db8::1"), net.ParseIP("2001:db8::2"),
			net.ParseIP("127.0.0.1"), net.ParseIP("::1"),
		},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		return nil, tls.Certificate{}, err
	}
	keyDER, err := x509.MarshalECPrivateKey(leafKey)
	if err != nil {
		return nil, tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	leafPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	cert, err = tls.X509KeyPair(leafPEM, keyPEM)
	if err != nil {
		return nil, tls.Certificate{}, err
	}
	caPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	return caPEM, cert, nil
}
