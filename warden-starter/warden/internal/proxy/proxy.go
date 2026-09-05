// Package proxy implements Warden's small, policy-enforcing HTTP proxy.
// DNS lookup happens only after a destination has passed the hostname
// allowlist, so denied names are never sent to a resolver.
package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/warden-sandbox/warden/internal/audit"
)

// Server enforces Warden's hostname allowlist. It listens on a Unix socket
// for the Linux and macOS backends (the socket is bind-mounted into the
// sandbox's network namespace) or on a loopback TCP address for the Windows
// backend (the only destination that backend's WFP filters permit).
type Server struct {
	listener net.Listener
	path     string // Unix socket path, empty for TCP listeners
	allow    map[string]struct{}
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
		s.allow[strings.ToLower(host)] = struct{}{}
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
		s.allow[strings.ToLower(host)] = struct{}{}
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
	if err := (&http.Server{Handler: s}).Serve(&oneConnListener{conn: conn}); err != nil && err != http.ErrServerClosed {
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
		s.deny(w, r.Method, net.JoinHostPort(host, port), "host is not in network.allow")
		return
	}

	// net.Dial performs DNS resolution only after allowed(host), which is the
	// crucial ordering that prevents blocked hostnames leaking in DNS queries.
	upstream, err := (&net.Dialer{}).DialContext(r.Context(), "tcp", net.JoinHostPort(host, port))
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
	_, ok := s.allow[strings.ToLower(host)]
	return ok
}

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

// oneConnListener lets net/http parse exactly one proxied connection.
type oneConnListener struct {
	conn net.Conn
	once sync.Once
}

func (l *oneConnListener) Accept() (net.Conn, error) {
	var c net.Conn
	l.once.Do(func() { c = l.conn })
	if c == nil {
		return nil, io.EOF
	}
	return c, nil
}
func (l *oneConnListener) Close() error   { return nil }
func (l *oneConnListener) Addr() net.Addr { return l.conn.LocalAddr() }

// ValidateURL exists solely to keep URL parsing behavior testable for
// callers that construct proxy requests programmatically.
func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid proxy URL %q", raw)
	}
	return nil
}
