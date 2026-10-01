package mcpbridge

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Remote struct {
	url     string
	client  *http.Client
	token   string
	session string
	limit   int
}

func PublicAddress(ip net.IP) bool {
	return ip != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsUnspecified() &&
		!net.ParseIP("100.64.0.0").Equal(ip) && !inCIDR(ip, "100.64.0.0/10") && !inCIDR(ip, "192.0.0.0/24") && !inCIDR(ip, "198.18.0.0/15") && !inCIDR(ip, "2001:db8::/32") && !inCIDR(ip, "192.0.2.0/24") && !inCIDR(ip, "198.51.100.0/24") && !inCIDR(ip, "203.0.113.0/24")
}
func inCIDR(ip net.IP, s string) bool { _, n, _ := net.ParseCIDR(s); return n.Contains(ip) }
func NewRemote(endpoint, token string, allowLoopback bool, limit int) (*Remote, error) {
	u, e := url.Parse(endpoint)
	if e != nil || u.Host == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" {
		return nil, fmt.Errorf("upstream requires an explicit URL without credentials, query or fragment")
	}
	loop := net.ParseIP(u.Hostname())
	isLoop := loop != nil && loop.IsLoopback()
	if u.Scheme != "https" && !(allowLoopback && isLoop && u.Scheme == "http") {
		return nil, fmt.Errorf("HTTPS required; explicit loopback HTTP opt-in only")
	}
	tr := &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, ResponseHeaderTimeout: 30 * time.Second}
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(addr)
		if e != nil {
			return nil, e
		}
		ips, e := net.DefaultResolver.LookupIPAddr(ctx, host)
		if e != nil {
			return nil, e
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no upstream address")
		}
		for _, ip := range ips {
			if !PublicAddress(ip.IP) && !(allowLoopback && isLoop && ip.IP.IsLoopback()) {
				return nil, fmt.Errorf("upstream resolved to forbidden address")
			}
		}
		// Dial the checked IP, preserving hostname for TLS; a second DNS
		// lookup cannot rebind the connection to an internal service.
		return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
	return &Remote{url: endpoint, token: token, limit: limit, client: &http.Client{Transport: tr, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("upstream redirects refused") }}}, nil
}
func (r *Remote) Close() error { r.client.CloseIdleConnections(); return nil }
func (r *Remote) request(ctx context.Context, m Message) (Message, error) {
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, r.url, bytes.NewReader(Encode(m)))
	if e != nil {
		return Message{}, e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", Protocol)
	if r.session != "" {
		req.Header.Set("Mcp-Session-Id", r.session)
	}
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}
	resp, e := r.client.Do(req)
	if e != nil {
		return Message{}, fmt.Errorf("upstream HTTP failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 202 && len(m.ID) == 0 {
		return Message{}, nil
	}
	if resp.StatusCode != 200 {
		return Message{}, fmt.Errorf("upstream HTTP status %d", resp.StatusCode)
	}
	if m.Method == "initialize" {
		r.session = resp.Header.Get("Mcp-Session-Id")
	}
	var b []byte
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		s := bufio.NewScanner(io.LimitReader(resp.Body, int64(r.limit)+1))
		s.Buffer(make([]byte, 4096), r.limit)
		var data []string
		size := 0
		for s.Scan() {
			line := s.Text()
			size += len(line) + 1
			if size > r.limit {
				return Message{}, fmt.Errorf("upstream stream too large")
			}
			if line == "" && len(data) > 0 {
				raw := []byte(strings.Join(data, "\n"))
				out, e := Parse(raw)
				data = nil
				if e != nil {
					return Message{}, e
				}
				if out.Method != "" {
					if len(out.ID) > 0 {
						return Message{}, fmt.Errorf("server requests unsupported")
					}
					continue
				}
				if string(out.ID) != string(m.ID) {
					return Message{}, fmt.Errorf("unsolicited upstream response")
				}
				return out, nil
			}
			if strings.HasPrefix(line, "data:") {
				data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			}
		}
		return Message{}, fmt.Errorf("upstream stream ended without a response")
	} else if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		return Message{}, fmt.Errorf("unsupported upstream content type")
	}
	b, e = io.ReadAll(io.LimitReader(resp.Body, int64(r.limit)+1))
	if e != nil || len(b) > r.limit {
		return Message{}, fmt.Errorf("upstream response too large")
	}
	out, e := Parse(b)
	if e != nil || out.Method != "" || string(out.ID) != string(m.ID) {
		return Message{}, fmt.Errorf("invalid upstream response")
	}
	return out, nil
}
func (r *Remote) RoundTrip(ctx context.Context, m Message) (Message, error) { return r.request(ctx, m) }
func (r *Remote) Notify(ctx context.Context, m Message) error               { _, e := r.request(ctx, m); return e }
