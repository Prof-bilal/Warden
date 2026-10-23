// Package proxy implements Warden's small, policy-enforcing HTTP proxy.
// DNS lookup happens only after a destination has passed the hostname
// allowlist, so denied names are never sent to a resolver.
package proxy

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/warden-sandbox/warden/internal/audit"
)

// Decision is what an Approver resolves for one blocked network request.
// Deny keeps the current behavior (403 + audit event). The allow variants
// all let the current request through; they differ in what is remembered.
type Decision int

const (
	// Deny blocks the request. This is also the fail-closed outcome when no
	// approver is set, the prompter has no terminal, or approval times out.
	Deny Decision = iota
	// AllowOnce lets this request through without remembering anything: the
	// next request to the host prompts again.
	AllowOnce
	// AllowSession remembers the host in memory for the rest of this run.
	AllowSession
	// AllowAndSave remembers the host for this run; the approver is also
	// expected to persist it to the policy file before returning.
	AllowAndSave
)

const (
	// tlsRecordHandshake is the TLS record type for a handshake. A CONNECT
	// tunnel whose first client byte is something else is forwarded without
	// inspection (for example a plain WebSocket or a server-first protocol).
	tlsRecordHandshake = 0x16
	// tlsMinVersion is TLS 1.2. Warden documents 1.2 as the minimum for
	// CONNECT tunnels and rejects a ClientHello whose highest offered
	// version is lower.
	tlsMinVersion = 0x0303
	// tlsInspectTimeout bounds how long the proxy waits for the first client
	// bytes before treating the tunnel as non-TLS/server-first. It does not
	// delay the upstream-to-client direction.
	tlsInspectTimeout = 2 * time.Second
	// maxTLSRecordLen is the largest TLS plaintext record the proxy will
	// buffer for ClientHello inspection (16 KiB plus the allowed expansion).
	maxTLSRecordLen = 1<<14 + 2048
)

var errTLSFloor = errors.New("TLS ClientHello offers only versions below 1.2")

// Approver resolves one blocked request to host:port. It is called
// synchronously in the request path, so it may prompt the user; concurrent
// requests serialize inside the approver implementation, not here.
type Approver func(host, port string) Decision

// Server enforces Warden's hostname allowlist. It listens on a Unix socket
// for the Linux and macOS backends (the socket is bind-mounted into the
// sandbox's network namespace) or on a loopback TCP address for the Windows
// backend (the only destination that backend's WFP filters permit).
type Server struct {
	listener  net.Listener
	path      string // Unix socket path, empty for TCP listeners
	mu        sync.RWMutex
	allow     map[string]struct{}            // host patterns granted on every port
	allowPort map[string]map[string]struct{} // host pattern -> explicitly granted ports
	approver  Approver
	audit     *audit.Logger
	closed    sync.Once
}

func Start(allow []string, logger *audit.Logger) (*Server, error) {
	dir, err := os.MkdirTemp("", "warden-proxy-")
	if err != nil {
		return nil, fmt.Errorf("create proxy directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("protect proxy directory: %w", err)
	}
	path := filepath.Join(dir, "egress.sock")
	l, err := net.Listen("unix", path)
	if err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("listen on proxy socket: %w", err)
	}
	s := &Server{
		listener:  l,
		path:      path,
		allow:     make(map[string]struct{}),
		allowPort: make(map[string]map[string]struct{}),
		audit:     logger,
	}
	if err := s.addAllowRules(allow); err != nil {
		_ = l.Close()
		_ = os.RemoveAll(dir)
		return nil, err
	}
	go s.serve()
	return s, nil
}

// StartTCP starts the policy proxy on a loopback TCP address instead of a
// Unix socket. The Windows backend uses this: AppContainer processes have
// no meaningful Unix-socket story, while WFP filters can hard-permit exactly
// this loopback endpoint and block everything else outbound.
func StartTCP(allow []string, logger *audit.Logger) (*Server, error) {
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen on loopback proxy: %w", err)
	}
	s := &Server{
		listener:  l,
		allow:     make(map[string]struct{}),
		allowPort: make(map[string]map[string]struct{}),
		audit:     logger,
	}
	if err := s.addAllowRules(allow); err != nil {
		_ = l.Close()
		return nil, err
	}
	go s.serve()
	return s, nil
}

// addAllowRules parses network.allow entries. A bare hostname or IP grants
// every port; host:port grants only that port. IPv6 with a port must be
// bracketed. Entries are validated by policy.Load before they reach here,
// but Start is also a public seam, so reject malformed rules instead of
// silently widening them to every port.
func (s *Server) addAllowRules(rules []string) error {
	for _, rule := range rules {
		host, port, hasPort, err := splitAllowRule(rule)
		if err != nil {
			return fmt.Errorf("proxy allow rule: %w", err)
		}
		host = normalizeHost(host)
		if !hasPort {
			s.allow[host] = struct{}{}
			continue
		}
		if s.allowPort[host] == nil {
			s.allowPort[host] = make(map[string]struct{})
		}
		s.allowPort[host][port] = struct{}{}
	}
	return nil
}

func splitAllowRule(rule string) (host, port string, hasPort bool, err error) {
	if rule == "" {
		return "", "", false, fmt.Errorf("empty host")
	}
	if strings.HasPrefix(rule, "[") {
		host, port, err = net.SplitHostPort(rule)
		if err != nil || host == "" || port == "" {
			return "", "", false, fmt.Errorf("%q is not a valid [IPv6]:port", rule)
		}
	} else if strings.Count(rule, ":") == 1 {
		host, port, err = net.SplitHostPort(rule)
		if err != nil || host == "" || port == "" {
			return "", "", false, fmt.Errorf("%q is not a bare host, IP, or host:port", rule)
		}
	} else {
		if strings.Contains(rule, ":") && net.ParseIP(rule) == nil {
			return "", "", false, fmt.Errorf("%q is not a bare host, IP, or host:port", rule)
		}
		return rule, "", false, nil
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != port {
		return "", "", false, fmt.Errorf("%q has an invalid port", rule)
	}
	return host, port, true, nil
}

// Addr returns the TCP listen address (host:port) for a TCP listener. It is
// empty for Unix-socket listeners; use SocketPath instead for those.
func (s *Server) Addr() string {
	if s.path != "" {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *Server) SocketPath() string { return s.path }

func (s *Server) Close() error {
	var err error
	s.closed.Do(func() {
		err = s.listener.Close()
		if s.path != "" {
			err2 := os.RemoveAll(filepath.Dir(s.path))
			if err == nil {
				err = err2
			}
		}
	})
	return err
}

func (s *Server) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.serveConn(conn)
	}
}

func (s *Server) serveConn(conn net.Conn) {
	defer conn.Close()
	// A bridge transports one sandbox TCP connection verbatim. HTTP proxy
	// requests are then parsed and authorized here, outside the namespace.
	wrapped := &closeNotifyingConn{Conn: conn, done: make(chan struct{})}
	if err := (&http.Server{Handler: s, ReadHeaderTimeout: 30 * time.Second}).Serve(&oneConnListener{conn: wrapped, done: wrapped.done}); err != nil && err != http.ErrServerClosed {
		return
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, port, err := destination(r)
	if err != nil {
		s.deny(w, r.Method, r.Host, "invalid proxy destination")
		return
	}
	if !s.allowed(host, port) {
		// A host that is allowlisted only on other ports is a deliberate
		// port restriction, not an unknown host. Do not offer interactive
		// approval here: approving would otherwise widen the grant to every
		// port for the rest of the run.
		if s.hostKnown(host) {
			s.deny(w, r.Method, net.JoinHostPort(host, port), "host is not allowed on this port")
			return
		}
		// Interactive approval mode (M7): give the user one chance to
		// allow the host before falling back to the deny path. Without
		// an approver this block is skipped and the request is denied.
		if s.approverFor(host, port) {
			s.proxyRequest(w, r, host, port)
			return
		}
		s.deny(w, r.Method, net.JoinHostPort(host, port), "host is not in network.allow")
		return
	}

	s.proxyRequest(w, r, host, port)
}

// approverFor asks the approver about host:port. It reports whether the
// current request may proceed. Session/persistent approvals are recorded in
// the allowlist so later requests pass without re-prompting; AllowOnce
// proceeds without recording.
func (s *Server) approverFor(host, port string) bool {
	s.mu.RLock()
	approver := s.approver
	s.mu.RUnlock()
	if approver == nil {
		return false
	}
	switch approver(host, port) {
	case AllowOnce:
		s.log("connect", net.JoinHostPort(host, port), true, "approved by user for this request only")
		return true
	case AllowSession:
		s.Grant(host)
		s.log("connect", net.JoinHostPort(host, port), true, "approved by user for this session")
		return true
	case AllowAndSave:
		s.Grant(host)
		s.log("connect", net.JoinHostPort(host, port), true, "approved by user and saved to policy")
		return true
	default:
		return false
	}
}

// proxyRequest dials an approved upstream and forwards the request.
// net.Dial performs DNS resolution only after the allow check, which is the
// crucial ordering that prevents blocked hostnames leaking in DNS queries.
func (s *Server) proxyRequest(w http.ResponseWriter, r *http.Request, host, port string) {
	upstream, err := dialApproved(r.Context(), host, port)
	if err != nil {
		s.log("connect", net.JoinHostPort(host, port), false, err.Error())
		http.Error(w, "Warden proxy could not reach allowed host", http.StatusBadGateway)
		return
	}
	defer upstream.Close()
	s.log("connect", net.JoinHostPort(host, port), true, "host allowed")

	if r.Method == http.MethodConnect {
		h, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "proxy connection cannot be hijacked", http.StatusInternalServerError)
			return
		}
		client, _, err := h.Hijack()
		if err != nil {
			return
		}
		defer client.Close()
		if _, err := io.WriteString(client, "HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
			return
		}
		if err := proxyTunnel(client, upstream); err != nil {
			s.log("connect", net.JoinHostPort(host, port), false, err.Error())
		}
		return
	}

	// Ordinary HTTP proxy request.  The request-target must be absolute;
	// force an origin-form request to the approved upstream.
	r.RequestURI = ""
	r.URL.Scheme = ""
	r.URL.Host = ""
	stripHopByHopHeaders(r.Header)
	if err := r.Write(upstream); err != nil {
		return
	}
	resp, err := http.ReadResponse(bufio.NewReader(upstream), r)
	if err != nil {
		http.Error(w, "invalid response from upstream", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) allowed(host, port string) bool {
	host = normalizeHost(host)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.allowedLocked(host, port)
}

func (s *Server) allowedLocked(host, port string) bool {
	if _, ok := s.allow[host]; ok {
		return true
	}
	if ports := s.allowPort[host]; ports != nil {
		if _, ok := ports[port]; ok {
			return true
		}
	}
	// A wildcard covers subdomains, never its apex.
	for suffix := host; ; {
		i := strings.IndexByte(suffix, '.')
		if i < 0 {
			return false
		}
		suffix = suffix[i+1:]
		if host == suffix {
			continue
		}
		if _, ok := s.allow["*."+suffix]; ok {
			return true
		}
		if ports := s.allowPort["*."+suffix]; ports != nil {
			if _, ok := ports[port]; ok {
				return true
			}
		}
	}
}

// hostKnown reports whether the host appears in the allowlist for at least
// one port. It is used to distinguish a deliberate port restriction from an
// unknown host that may be offered interactive approval.
func (s *Server) hostKnown(host string) bool {
	host = normalizeHost(host)
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.allow[host]; ok {
		return true
	}
	if _, ok := s.allowPort[host]; ok {
		return true
	}
	for suffix := host; ; {
		i := strings.IndexByte(suffix, '.')
		if i < 0 {
			return false
		}
		suffix = suffix[i+1:]
		if host == suffix {
			continue
		}
		if _, ok := s.allow["*."+suffix]; ok {
			return true
		}
		if _, ok := s.allowPort["*."+suffix]; ok {
			return true
		}
	}
}

// dialApproved resolves first, then refuses private, loopback, link-local,
// multicast, and unspecified addresses before dialing an individual IP. This
// prevents an allowlisted DNS name from rebinding to an internal service.
// An explicit IP literal (127.0.0.1, 10.x, etc.) is already allowlisted by
// name, so it is dialed directlyonly unspecified/multicast are still
// rejected as they can never be valid upstreams.
func dialApproved(ctx context.Context, host, port string) (net.Conn, error) {
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsUnspecified() || ip.IsMulticast() {
			return nil, fmt.Errorf("destination resolves to a restricted address")
		}
		return (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), port))
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, addr := range addrs {
		if forbiddenIP(addr.IP) {
			lastErr = fmt.Errorf("destination resolves to a restricted address")
			continue
		}
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(addr.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("destination has no usable addresses")
}

func forbiddenIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsPrivate()
}

// SetApprover installs the interactive-approval callback (M7). A nil
// approver restores plain deny-by-default behavior.
func (s *Server) SetApprover(a Approver) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.approver = a
}

// Grant adds host to the in-memory allowlist for the rest of this run.
// Persistent approvals additionally write through to the policy file via the
// approver; Grant itself never touches the filesystem.
func (s *Server) Grant(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allow[normalizeHost(host)] = struct{}{}
}

func normalizeHost(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}

// IsAllowed reports whether host is currently allowlisted (policy grants
// plus any session approvals so far).
func (s *Server) IsAllowed(host string) bool { return s.hostKnown(host) }

func (s *Server) deny(w http.ResponseWriter, action, resource, reason string) {
	s.log(action, resource, false, reason)
	http.Error(w, "Warden blocked network destination", http.StatusForbidden)
}

func (s *Server) log(action, resource string, allowed bool, reason string) {
	_ = s.audit.Log(audit.Event{Type: "network", Action: action, Resource: resource, Allowed: allowed, Reason: reason})
}

func destination(r *http.Request) (host, port string, err error) {
	authority := r.Host
	if r.Method != http.MethodConnect {
		if r.URL == nil || r.URL.Host == "" {
			return "", "", fmt.Errorf("HTTP proxy request has no absolute URL")
		}
		authority = r.URL.Host
	}
	host, port, err = net.SplitHostPort(authority)
	if err == nil {
		if host == "" || port == "" {
			return "", "", fmt.Errorf("empty host or port")
		}
		return host, port, nil
	}
	if r.Method == http.MethodConnect && !strings.Contains(authority, ":") {
		return authority, "443", nil
	}
	if strings.Contains(authority, ":") { // malformed or unbracketed IPv6
		return "", "", fmt.Errorf("invalid host:port %q", authority)
	}
	if r.URL != nil && r.URL.Scheme == "https" {
		return authority, "443", nil
	}
	return authority, "80", nil
}

// proxyTunnel relays a CONNECT tunnel in both directions. The client-to-
// upstream direction inspects the first TLS record when the client starts a
// TLS handshake; the upstream-to-client direction is never delayed, so
// server-first protocols still work.
func proxyTunnel(client, upstream net.Conn) error {
	br := bufio.NewReaderSize(client, 64*1024)
	done := make(chan error, 2)
	go func() {
		_, err := io.Copy(client, upstream)
		done <- err
	}()
	go func() {
		done <- copyClientTunnel(client, br, upstream)
	}()
	return <-done
}

func copyClientTunnel(client net.Conn, br *bufio.Reader, upstream net.Conn) error {
	_ = client.SetReadDeadline(time.Now().Add(tlsInspectTimeout))
	first, err := br.Peek(1)
	_ = client.SetReadDeadline(time.Time{})
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			// The client has not sent anything yet. This is normal for
			// server-first protocols; relay the upstream greeting and any
			// later client bytes without a TLS inspection.
			_, copyErr := io.Copy(upstream, br)
			return copyErr
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("inspect CONNECT client: %w", err)
	}
	if first[0] != tlsRecordHandshake {
		_, copyErr := io.Copy(upstream, br)
		return copyErr
	}

	// The client started a TLS handshake. Read the complete ClientHello
	// record before forwarding anything so a sub-1.2 client cannot reach the
	// upstream. A slow/fragmented handshake is bounded by the same deadline.
	_ = client.SetReadDeadline(time.Now().Add(tlsInspectTimeout))
	defer client.SetReadDeadline(time.Time{})
	header, err := br.Peek(5)
	if err != nil {
		return fmt.Errorf("inspect TLS record header: %w", err)
	}
	length := int(binary.BigEndian.Uint16(header[3:5]))
	if length < 4 || length > maxTLSRecordLen {
		return fmt.Errorf("invalid TLS record length %d", length)
	}
	record, err := br.Peek(5 + length)
	if err != nil {
		return fmt.Errorf("inspect TLS ClientHello: %w", err)
	}
	if !clientHelloAllowsTLS12(record) {
		// Send a fatal protocol_version alert using the client's record
		// version so the client gets an explicit TLS error rather than a
		// silent EOF. The upstream never receives the ClientHello.
		alert := []byte{0x15, header[1], header[2], 0x00, 0x02, 0x02, 0x46}
		_, _ = client.Write(alert)
		return errTLSFloor
	}
	_ = client.SetReadDeadline(time.Time{})
	_, copyErr := io.Copy(upstream, br)
	return copyErr
}

// clientHelloAllowsTLS12 reports whether a TLS ClientHello offers TLS 1.2 or
// newer. It deliberately fails closed on malformed handshakes. The legacy
// ClientHello version is checked first; when the supported_versions extension
// is present it is authoritative (TLS 1.3 fixes legacy_version at 1.2).
func clientHelloAllowsTLS12(record []byte) bool {
	if len(record) < 9 || record[0] != tlsRecordHandshake {
		return false
	}
	body := record[5:]
	if len(body) < 4 || body[0] != 0x01 { // ClientHello
		return false
	}
	handshakeLen := int(body[1])<<16 | int(body[2])<<8 | int(body[3])
	if handshakeLen < 34 || 4+handshakeLen > len(body) {
		return false
	}
	hs := body[4 : 4+handshakeLen]
	legacyVersion := binary.BigEndian.Uint16(hs[:2])

	pos := 34 // legacy_version(2) + random(32)
	if pos >= len(hs) {
		return legacyVersion >= tlsMinVersion
	}
	sessionIDLen := int(hs[pos])
	pos++
	if pos+sessionIDLen > len(hs) {
		return false
	}
	pos += sessionIDLen
	if pos+2 > len(hs) {
		return false
	}
	cipherSuitesLen := int(binary.BigEndian.Uint16(hs[pos:]))
	pos += 2
	if pos+cipherSuitesLen > len(hs) {
		return false
	}
	pos += cipherSuitesLen
	if pos >= len(hs) {
		return legacyVersion >= tlsMinVersion
	}
	compressionLen := int(hs[pos])
	pos++
	if pos+compressionLen > len(hs) {
		return false
	}
	pos += compressionLen
	if pos == len(hs) {
		return legacyVersion >= tlsMinVersion
	}
	if pos+2 > len(hs) {
		return false
	}
	extensionsLen := int(binary.BigEndian.Uint16(hs[pos:]))
	pos += 2
	if pos+extensionsLen > len(hs) {
		return false
	}
	extensions := hs[pos : pos+extensionsLen]

	var maxVersion uint16
	for len(extensions) >= 4 {
		extType := binary.BigEndian.Uint16(extensions[:2])
		extLen := int(binary.BigEndian.Uint16(extensions[2:4]))
		extensions = extensions[4:]
		if extLen > len(extensions) {
			return false
		}
		data := extensions[:extLen]
		extensions = extensions[extLen:]
		if extType != 0x002b { // supported_versions
			continue
		}
		if len(data) < 1 {
			return false
		}
		n := int(data[0])
		if n%2 != 0 || 1+n > len(data) {
			return false
		}
		for i := 1; i < 1+n; i += 2 {
			if v := binary.BigEndian.Uint16(data[i:]); v > maxVersion {
				maxVersion = v
			}
		}
	}
	if maxVersion != 0 {
		return maxVersion >= tlsMinVersion
	}
	return legacyVersion >= tlsMinVersion
}

func copyHeader(dst, src http.Header) {
	for k, values := range src {
		for _, value := range values {
			dst.Add(k, value)
		}
	}
}

// stripHopByHopHeaders removes headers that apply only to the client-proxy
// hop before a request is forwarded upstream. Proxy-Authorization must never
// leave the proxy: a sandboxed process could otherwise plant credentials for
// an upstream service, and a compromised proxy credential must not be
// replayed to the destination. The Connection header can also name additional
// hop-by-hop fields, so remove those before deleting Connection itself.
func stripHopByHopHeaders(h http.Header) {
	for _, value := range h.Values("Connection") {
		for _, token := range strings.Split(value, ",") {
			if token = strings.TrimSpace(token); token != "" {
				h.Del(token)
			}
		}
	}
	for _, name := range []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Proxy-Connection",
		"TE",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	} {
		h.Del(name)
	}
}

// oneConnListener lets net/http parse exactly one proxied connection. The
// second Accept blocks until that connection has actually been finished —
// closed by the server (plain requests) or by the hijacking handler
// (CONNECT)so Serve does not return and trigger the caller's deferred
// close while the handler goroutine is still relaying data. Without this,
// the request context is cancelled mid-handler and an allowed proxied
// request can fail with an empty reply.
type oneConnListener struct {
	conn net.Conn
	done <-chan struct{}
	once sync.Once
}

func (l *oneConnListener) Accept() (net.Conn, error) {
	var c net.Conn
	l.once.Do(func() { c = l.conn })
	if c == nil {
		<-l.done
		return nil, io.EOF
	}
	return c, nil
}
func (l *oneConnListener) Close() error   { return nil }
func (l *oneConnListener) Addr() net.Addr { return l.conn.LocalAddr() }

// closeNotifyingConn signals done exactly once when the connection is
// closed, whether by net/http or by the hijacking handler.
type closeNotifyingConn struct {
	net.Conn
	once sync.Once
	done chan struct{}
}

func (c *closeNotifyingConn) Close() error {
	c.once.Do(func() { close(c.done) })
	return c.Conn.Close()
}

// ValidateURL exists solely to keep URL parsing behavior testable for
// callers that construct proxy requests programmatically.
func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid proxy URL %q", raw)
	}
	return nil
}
