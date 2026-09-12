// Package proxy implements Warden's small, policy-enforcing HTTP proxy.
// DNS lookup happens only after a destination has passed the hostname
// allowlist, so denied names are never sent to a resolver.
package proxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

// Approver resolves one blocked request to host:port. It is called
// synchronously in the request path, so it may prompt the user; concurrent
// requests serialize inside the approver implementation, not here.
type Approver func(host, port string) Decision

// Server enforces Warden's hostname allowlist. It listens on a Unix socket
// for the Linux and macOS backends (the socket is bind-mounted into the
// sandbox's network namespace) or on a loopback TCP address for the Windows
// backend (the only destination that backend's WFP filters permit).
type Server struct {
	listener net.Listener
	path     string // Unix socket path, empty for TCP listeners
	mu       sync.RWMutex
	allow    map[string]struct{}
	approver Approver
	audit    *audit.Logger
	closed   sync.Once
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
	s := &Server{listener: l, path: path, allow: make(map[string]struct{}), audit: logger}
	for _, host := range allow {
		s.allow[normalizeHost(host)] = struct{}{}
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
	s := &Server{listener: l, allow: make(map[string]struct{}), audit: logger}
	for _, host := range allow {
		s.allow[normalizeHost(host)] = struct{}{}
	}
	go s.serve()
	return s, nil
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
	if !s.allowed(host) {
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
		proxyBoth(client, upstream)
		return
	}

	// Ordinary HTTP proxy request.  The request-target must be absolute;
	// force an origin-form request to the approved upstream.
	r.RequestURI = ""
	r.URL.Scheme = ""
	r.URL.Host = ""
	r.Header.Del("Proxy-Connection")
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

func (s *Server) allowed(host string) bool {
	host = normalizeHost(host)
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.allow[host]; ok {
		return true
	}
	// A wildcard covers subdomains, never its apex.
	for suffix := host; ; {
		i := strings.IndexByte(suffix, '.')
		if i < 0 {
			return false
		}
		suffix = suffix[i+1:]
		if _, ok := s.allow["*."+suffix]; ok && host != suffix {
			return true
		}
	}
}

// dialApproved resolves first, then refuses private, loopback, link-local,
// multicast, and unspecified addresses before dialing an individual IP. This
// prevents an allowlisted DNS name from rebinding to an internal service.
// An explicit IP literal (127.0.0.1, 10.x, etc.) is already allowlisted by
// name, so it is dialed directly — only unspecified/multicast are still
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
func (s *Server) IsAllowed(host string) bool { return s.allowed(host) }

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

func proxyBoth(a, b net.Conn) {
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(a, b); done <- struct{}{} }()
	go func() { _, _ = io.Copy(b, a); done <- struct{}{} }()
	<-done
}

func copyHeader(dst, src http.Header) {
	for k, values := range src {
		for _, value := range values {
			dst.Add(k, value)
		}
	}
}

// oneConnListener lets net/http parse exactly one proxied connection. The
// second Accept blocks until that connection has actually been finished —
// closed by the server (plain requests) or by the hijacking handler
// (CONNECT) — so Serve does not return and trigger the caller's deferred
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
