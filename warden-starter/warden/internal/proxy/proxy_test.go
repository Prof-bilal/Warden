package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/warden-sandbox/warden/internal/audit"
)

func TestDestination(t *testing.T) {
	cases := []struct {
		name, method, raw, host, wantHost, wantPort string
	}{
		{"connect", http.MethodConnect, "", "api.example.test:8443", "api.example.test", "8443"},
		{"http default port", http.MethodGet, "http://api.example.test/path", "", "api.example.test", "80"},
		{"https default port", http.MethodGet, "https://api.example.test/path", "", "api.example.test", "443"},
		{"connect default port", http.MethodConnect, "", "api.example.test", "api.example.test", "443"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.raw, nil)
			if tc.host != "" {
				r.Host = tc.host
			}
			host, port, err := destination(r)
			if err != nil || host != tc.wantHost || port != tc.wantPort {
				t.Fatalf("destination = %q, %q, %v", host, port, err)
			}
		})
	}
}

func TestAllowedNormalizesAndMatchesWildcards(t *testing.T) {
	s := &Server{allow: map[string]struct{}{"*.github.com": {}, "api.example.com": {}}}
	for _, host := range []string{"api.github.com", "A.B.GITHUB.COM", "API.EXAMPLE.COM."} {
		if !s.allowed(host, "443") {
			t.Errorf("%q should be allowed", host)
		}
	}
	for _, host := range []string{"github.com", "notgithub.com"} {
		if s.allowed(host, "443") {
			t.Errorf("%q should not be allowed", host)
		}
	}
}

func TestAllowedHonorsPortSpecificGrants(t *testing.T) {
	s := &Server{
		allow: map[string]struct{}{"any.example.com": {}},
		allowPort: map[string]map[string]struct{}{
			"api.example.com": {"443": {}},
			"*.github.com":    {"8443": {}},
		},
	}
	cases := []struct {
		host, port string
		want       bool
		known      bool
	}{
		{"any.example.com", "80", true, true},
		{"any.example.com", "443", true, true},
		{"api.example.com", "443", true, true},
		{"api.example.com", "80", false, true},
		{"x.github.com", "8443", true, true},
		{"x.github.com", "443", false, true},
		{"github.com", "8443", false, false},
		{"unknown.example.com", "443", false, false},
	}
	for _, tc := range cases {
		if got := s.allowed(tc.host, tc.port); got != tc.want {
			t.Errorf("allowed(%q, %q) = %v, want %v", tc.host, tc.port, got, tc.want)
		}
		if got := s.hostKnown(tc.host); got != tc.known {
			t.Errorf("hostKnown(%q) = %v, want %v", tc.host, got, tc.known)
		}
	}
}

func TestForbiddenIP(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "169.254.1.1", "::1", "fc00::1"} {
		if !forbiddenIP(net.ParseIP(raw)) {
			t.Errorf("%s should be forbidden", raw)
		}
	}
	if forbiddenIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("public address should not be forbidden")
	}
}

func TestProxyApproverAllowSessionGrantsOnce(t *testing.T) {
	var log bytes.Buffer
	s := &Server{allow: map[string]struct{}{}, audit: audit.New(&log)}
	calls := 0
	s.SetApprover(func(host, port string) Decision {
		calls++
		return AllowSession
	})

	connect := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodConnect, "http://127.0.0.1:1", nil)
		r.Host = "127.0.0.1:1"
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}

	// No listener on 127.0.0.1:1, so an approved request fails at dial with
	// 502the point is it got past the allowlist, not that it connected.
	if w := connect(); w.Code != http.StatusBadGateway {
		t.Fatalf("first status = %d, want %d", w.Code, http.StatusBadGateway)
	}
	if w := connect(); w.Code != http.StatusBadGateway {
		t.Fatalf("second status = %d, want %d", w.Code, http.StatusBadGateway)
	}
	if calls != 1 {
		t.Fatalf("approver calls = %d, want 1 (session grant must stick)", calls)
	}
	if !s.IsAllowed("127.0.0.1") {
		t.Fatal("host should be allowlisted after session approval")
	}
}

func TestProxyApproverAllowOncePromptsEveryTime(t *testing.T) {
	var log bytes.Buffer
	s := &Server{allow: map[string]struct{}{}, audit: audit.New(&log)}
	calls := 0
	s.SetApprover(func(host, port string) Decision {
		calls++
		return AllowOnce
	})
	for i := 0; i < 2; i++ {
		r := httptest.NewRequest(http.MethodConnect, "http://127.0.0.1:1", nil)
		r.Host = "127.0.0.1:1"
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != http.StatusBadGateway {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusBadGateway)
		}
	}
	if calls != 2 {
		t.Fatalf("approver calls = %d, want 2 (once must not stick)", calls)
	}
	if s.IsAllowed("127.0.0.1") {
		t.Fatal("host must not be allowlisted after allow-once")
	}
}

func TestProxyApproverDenyKeepsDenying(t *testing.T) {
	var log bytes.Buffer
	s := &Server{allow: map[string]struct{}{}, audit: audit.New(&log)}
	s.SetApprover(func(host, port string) Decision { return Deny })
	r := httptest.NewRequest(http.MethodConnect, "http://blocked.example:443", nil)
	r.Host = "blocked.example:443"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}

	// Unsetting the approver restores plain deny-by-default.
	s.SetApprover(nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status after unset = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestAddAllowRulesRejectsMalformedPorts(t *testing.T) {
	s := &Server{allow: map[string]struct{}{}, allowPort: map[string]map[string]struct{}{}}
	for _, rule := range []string{"api.example.com:0", "api.example.com:65536", "api.example.com:https", "api.example.com:", "foo:bar:baz"} {
		if err := s.addAllowRules([]string{rule}); err == nil {
			t.Errorf("addAllowRules(%q) succeeded, want error", rule)
		}
	}
	if err := s.addAllowRules([]string{"api.example.com:443", "[2001:db8::1]:8443", "bare.example.com"}); err != nil {
		t.Fatalf("addAllowRules(valid) = %v", err)
	}
	if !s.allowed("api.example.com", "443") || s.allowed("api.example.com", "80") {
		t.Fatal("port-specific rule was not enforced")
	}
	if !s.allowed("2001:db8::1", "8443") || s.allowed("2001:db8::1", "443") {
		t.Fatal("IPv6 port-specific rule was not enforced")
	}
	if !s.allowed("bare.example.com", "12345") {
		t.Fatal("bare host rule should allow every port")
	}
}

func TestProxyPortSpecificGrantDeniesOtherPort(t *testing.T) {
	var upstreamHit bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHit = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	host, port, err := net.SplitHostPort(strings.TrimPrefix(upstream.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	s := &Server{
		allow:     map[string]struct{}{},
		allowPort: map[string]map[string]struct{}{host: {"1": {}}},
		audit:     audit.New(&log),
	}
	r := httptest.NewRequest(http.MethodGet, "http://"+net.JoinHostPort(host, port)+"/", nil)
	r.Host = net.JoinHostPort(host, port)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if upstreamHit {
		t.Fatal("request reached the upstream despite a port mismatch")
	}
}

func TestProxyStripsProxyAuthAndHopByHopHeaders(t *testing.T) {
	var upstreamHeader http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHeader = r.Header.Clone()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	host, port, err := net.SplitHostPort(strings.TrimPrefix(upstream.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	s := &Server{allow: map[string]struct{}{host: {}}, audit: audit.New(&log)}
	r := httptest.NewRequest(http.MethodGet, "http://"+net.JoinHostPort(host, port)+"/", nil)
	r.Host = net.JoinHostPort(host, port)
	r.Header.Set("Proxy-Authorization", "Basic d2FyZGVuLWVncmVzcy10ZXN0")
	r.Header.Set("Proxy-Connection", "keep-alive")
	r.Header.Set("Connection", "X-Hop")
	r.Header.Set("X-Hop", "must-not-leave-the-proxy")
	r.Header.Set("X-End-To-End", "preserved")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusNoContent, w.Body.String())
	}
	for _, name := range []string{"Proxy-Authorization", "Proxy-Connection", "Connection", "X-Hop"} {
		if got := upstreamHeader.Get(name); got != "" {
			t.Errorf("upstream received hop-by-hop header %s=%q", name, got)
		}
	}
	if got := upstreamHeader.Get("X-End-To-End"); got != "preserved" {
		t.Errorf("end-to-end header = %q, want preserved", got)
	}
}

func makeClientHelloRecord(legacy uint16, supported []uint16) []byte {
	hs := []byte{0x01, 0, 0, 0, byte(legacy >> 8), byte(legacy)}
	hs = append(hs, make([]byte, 32)...) // random
	hs = append(hs, 0)                   // session id length
	hs = append(hs, 0, 2, 0x13, 0x01)    // one cipher suite
	hs = append(hs, 1, 0)                // null compression
	if supported != nil {
		data := []byte{byte(2 * len(supported))}
		for _, v := range supported {
			data = append(data, byte(v>>8), byte(v))
		}
		ext := []byte{0x00, 0x2b, 0, 0}
		ext[2], ext[3] = byte(len(data)>>8), byte(len(data))
		ext = append(ext, data...)
		hs = append(hs, byte(len(ext)>>8), byte(len(ext)))
		hs = append(hs, ext...)
	}
	hs[1], hs[2], hs[3] = byte((len(hs)-4)>>16), byte((len(hs)-4)>>8), byte(len(hs)-4)
	record := []byte{0x16, 0x03, 0x01, 0, 0}
	record[3], record[4] = byte(len(hs)>>8), byte(len(hs))
	return append(record, hs...)
}

func TestClientHelloAllowsTLS12(t *testing.T) {
	cases := []struct {
		name      string
		legacy    uint16
		supported []uint16
		want      bool
	}{
		{"TLS 1.1 legacy only", 0x0302, nil, false},
		{"TLS 1.0 legacy only", 0x0301, nil, false},
		{"TLS 1.2 legacy only", 0x0303, nil, true},
		{"supported_versions caps at 1.1", 0x0303, []uint16{0x0301, 0x0302}, false},
		{"TLS 1.2 supported_versions", 0x0301, []uint16{0x0303}, true},
		{"TLS 1.3 supported_versions", 0x0303, []uint16{0x0303, 0x0304}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := clientHelloAllowsTLS12(makeClientHelloRecord(tc.legacy, tc.supported)); got != tc.want {
				t.Errorf("clientHelloAllowsTLS12 = %v, want %v", got, tc.want)
			}
		})
	}
	valid := makeClientHelloRecord(0x0303, nil)
	for _, cut := range []int{0, 5, 8, len(valid) - 1} {
		if clientHelloAllowsTLS12(valid[:cut]) {
			t.Errorf("truncated ClientHello (%d bytes) should fail closed", cut)
		}
	}
}

func TestProxyTunnelRejectsTLSBelow12(t *testing.T) {
	proxyClient, client := net.Pipe()
	proxyUpstream, upstream := net.Pipe()
	defer proxyClient.Close()
	defer client.Close()
	defer proxyUpstream.Close()
	defer upstream.Close()

	done := make(chan error, 1)
	go func() { done <- proxyTunnel(proxyClient, proxyUpstream) }()
	go func() { _, _ = client.Write(makeClientHelloRecord(0x0302, nil)) }()

	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	alert := make([]byte, 7)
	if _, err := io.ReadFull(client, alert); err != nil {
		t.Fatalf("read TLS alert: %v", err)
	}
	if alert[0] != 0x15 || alert[5] != 0x02 || alert[6] != 0x46 {
		t.Fatalf("alert = % x, want fatal protocol_version", alert)
	}
	if err := <-done; err != errTLSFloor {
		t.Fatalf("proxyTunnel = %v, want errTLSFloor", err)
	}
	_ = upstream.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	var b [1]byte
	if _, err := upstream.Read(b[:]); err == nil {
		t.Fatal("sub-1.2 ClientHello reached the upstream")
	}
}

func TestProxyTunnelForwardsTLS12AndNonTLS(t *testing.T) {
	for _, tc := range []struct {
		name  string
		first []byte
	}{
		{"TLS 1.2", makeClientHelloRecord(0x0303, nil)},
		{"plain HTTP", []byte("GET / HTTP/1.1\r\nHost: example.test\r\n\r\n")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proxyClient, client := net.Pipe()
			proxyUpstream, upstream := net.Pipe()
			defer proxyClient.Close()
			defer client.Close()
			defer proxyUpstream.Close()
			defer upstream.Close()

			done := make(chan error, 1)
			go func() { done <- proxyTunnel(proxyClient, proxyUpstream) }()
			go func() { _, _ = client.Write(tc.first) }()

			_ = upstream.SetReadDeadline(time.Now().Add(2 * time.Second))
			got := make([]byte, len(tc.first))
			if _, err := io.ReadFull(upstream, got); err != nil {
				t.Fatalf("read forwarded bytes: %v", err)
			}
			if !bytes.Equal(got, tc.first) {
				t.Fatalf("forwarded bytes = % x, want % x", got, tc.first)
			}
			_ = upstream.Close()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("proxyTunnel did not return after upstream close")
			}
		})
	}
}

func TestProxyRejectsBeforeDialingOrDNS(t *testing.T) {
	var log bytes.Buffer
	s := &Server{allow: map[string]struct{}{"allowed.example": {}}, audit: audit.New(&log)}

	r := httptest.NewRequest(http.MethodConnect, "http://blocked.example:443", nil)
	r.Host = "blocked.example:443"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	var event audit.Event
	if err := json.Unmarshal([]byte(strings.TrimSpace(log.String())), &event); err != nil {
		t.Fatal(err)
	}
	if event.Allowed || event.Resource != "blocked.example:443" || !strings.Contains(event.Reason, "network.allow") {
		t.Errorf("event = %+v", event)
	}
}
