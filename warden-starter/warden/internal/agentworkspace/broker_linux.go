//go:build linux

package agentworkspace

import (
	"context"
	"crypto/tls"
	"encoding/base64"
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

// StartBroker exposes only fixed, pre-approved GET operations. No arbitrary
// commands or agent-controlled request bodies receive host credentials.
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
	path := filepath.Join(dir, ".warden", "broker.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, "", err
	}
	if err = os.Chmod(path, 0600); err != nil {
		listener.Close()
		return nil, "", err
	}
	transport := &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, DialContext: publicDial, ResponseHeaderTimeout: 10 * time.Second}
	client := &http.Client{Transport: transport, Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	handler := brokerHandler(profiles, secrets, logger, client)
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 25 * time.Second, IdleTimeout: 5 * time.Second, MaxHeaderBytes: 8192}
	go server.Serve(listener)
	return func() { server.Close(); transport.CloseIdleConnections(); os.Remove(path) }, path, nil
}

func brokerHandler(profiles map[string]policy.SecretProfile, secrets map[string]string, logger *audit.Logger, client *http.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/profiles/")
		profile, ok := profiles[name]
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
		// Exact-value echo defense; this is not protection against arbitrary encoded
		// or transformed responses from an untrusted upstream.
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
		// No upstream headers (cookies, redirects, or reflected credentials) escape.
		w.WriteHeader(res.StatusCode)
		_, _ = io.WriteString(w, text)
	})
}

func publicDial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || port != "443" {
		return nil, fmt.Errorf("HTTPS port required")
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("DNS failed")
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no public address")
	}
	for _, addr := range ips {
		if !publicIP(addr.IP) {
			return nil, fmt.Errorf("private/reserved address rejected")
		}
	}
	// Dial the checked address, avoiding a second DNS lookup.
	return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort(ips[0].IP.String(), port))
}
func publicIP(ip net.IP) bool {
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
	transport := &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	res, err := client.Get("http://warden/profiles/" + url.PathEscape(name))
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
