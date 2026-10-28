//go:build darwin || windows

package agentworkspace

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/policy"
)

// StartBroker returns the cleanup function and the supervisor-owned endpoint
// to put in WARDEN_AGENT_SOCKET. macOS uses a Unix socket so Seatbelt can
// permit exactly that file. Windows uses an ephemeral loopback port with a
// per-session bearer token; the AppContainer plan permits that one port.
func StartBroker(dir string, profiles map[string]policy.SecretProfile, logger *audit.Logger) (func(), string, error) {
	if logger == nil {
		return nil, "", fmt.Errorf("secret broker requires an audit logger")
	}
	secrets := map[string]string{}
	for name, p := range profiles {
		v := os.Getenv(p.Env)
		if v == "" {
			return nil, "", fmt.Errorf("secret profile %q has no credential", name)
		}
		if strings.ContainsAny(v, "\r\n") {
			return nil, "", fmt.Errorf("invalid credential header")
		}
		secrets[name] = v
	}

	var listener net.Listener
	var endpoint, sessionToken string
	var err error
	if isWindowsPortable() {
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, "", err
		}
		raw := make([]byte, 24)
		if _, err = rand.Read(raw); err != nil {
			listener.Close()
			return nil, "", err
		}
		sessionToken = hex.EncodeToString(raw)
		endpoint = "tcp://" + listener.Addr().String() + "/" + sessionToken
	} else {
		path := filepath.Join(dir, ".warden", "broker.sock")
		listener, err = net.Listen("unix", path)
		if err != nil {
			return nil, "", err
		}
		if err = os.Chmod(path, 0600); err != nil {
			listener.Close()
			return nil, "", err
		}
		endpoint = path
	}

	transport := &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, DialContext: publicDialPortable, ResponseHeaderTimeout: 10 * time.Second}
	client := &http.Client{Transport: transport, Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	handler := brokerHandlerPortable(profiles, secrets, logger, client, sessionToken)
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 25 * time.Second, IdleTimeout: 5 * time.Second, MaxHeaderBytes: 8192}
	go server.Serve(listener)
	return func() {
		server.Close()
		transport.CloseIdleConnections()
		if strings.HasPrefix(endpoint, "tcp://") {
			return
		}
		_ = os.Remove(endpoint)
	}, endpoint, nil
}

func isWindowsPortable() bool { return filepath.Separator == '\\' }

func brokerHandlerPortable(profiles map[string]policy.SecretProfile, secrets map[string]string, logger *audit.Logger, client *http.Client, sessionToken string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/profiles/")
		profile, ok := profiles[name]
		if sessionToken != "" && r.Header.Get("X-Warden-Agent-Session") != sessionToken {
			ok = false
		}
		if r.Method != "GET" || !strings.HasPrefix(r.URL.Path, "/profiles/") || r.URL.RawQuery != "" || r.ContentLength != 0 || !ok {
			_ = logger.Log(audit.Event{Type: "agent", Action: "secret_request", Resource: "unknown", Allowed: false, Reason: "profile request rejected"})
			http.Error(w, "profile request rejected", http.StatusForbidden)
			return
		}
		if err := logger.Log(audit.Event{Type: "agent", Action: "secret_request", Resource: name, Allowed: true, Reason: "fixed HTTPS GET approved before session"}); err != nil {
			http.Error(w, "audit unavailable", 500)
			return
		}
		req, err := http.NewRequestWithContext(r.Context(), "GET", profile.URL, nil)
		if err != nil {
			http.Error(w, "invalid profile", 500)
			return
		}
		req.Header.Set("Authorization", "Bearer "+secrets[name])
		res, err := client.Do(req)
		if err != nil {
			http.Error(w, "approved upstream unavailable", 502)
			return
		}
		defer res.Body.Close()
		body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20+1))
		if err != nil || len(body) > 2<<20 {
			http.Error(w, "upstream response exceeded limit", 502)
			return
		}
		text := string(body)
		for _, secret := range secrets {
			encoded, _ := json.Marshal(secret)
			variants := []string{secret, url.QueryEscape(secret), base64.StdEncoding.EncodeToString([]byte(secret)), string(encoded[1 : len(encoded)-1])}
			for _, v := range variants {
				if v != "" {
					text = strings.ReplaceAll(text, v, "[WARDEN_REDACTED]")
				}
			}
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(res.StatusCode)
		_, _ = io.WriteString(w, text)
	})
}

func publicDialPortable(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || port != "443" {
		return nil, fmt.Errorf("HTTPS port required")
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("DNS failed")
	}
	for _, addr := range ips {
		if !publicIPPortable(addr.IP) {
			return nil, fmt.Errorf("private/reserved address rejected")
		}
	}
	return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort(ips[0].IP.String(), port))
}

func publicIPPortable(ip net.IP) bool {
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, cidr := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32", "2001::/32", "2002::/16", "64:ff9b::/96"} {
		_, n, _ := net.ParseCIDR(cidr)
		if n.Contains(ip) {
			return false
		}
	}
	return true
}

func RequestProfile(socket, name string, out io.Writer) error {
	if name == "" || strings.ContainsAny(name, "/\\?#\r\n") {
		return fmt.Errorf("invalid profile name")
	}
	var network, address, token string
	if strings.HasPrefix(socket, "tcp://") {
		u, err := url.Parse(socket)
		if err != nil || u.Host == "" || strings.Trim(u.Path, "/") == "" {
			return fmt.Errorf("invalid Warden session broker")
		}
		network, address, token = "tcp", u.Host, strings.Trim(u.Path, "/")
	} else {
		network, address = "unix", socket
	}
	transport := &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", "http://warden/profiles/"+url.PathEscape(name), nil)
	if err != nil {
		return fmt.Errorf("invalid profile request")
	}
	if token != "" {
		req.Header.Set("X-Warden-Agent-Session", token)
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Warden session broker unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("approved profile failed (HTTP %d)", res.StatusCode)
	}
	_, err = io.Copy(out, io.LimitReader(res.Body, 2<<20))
	return err
}
