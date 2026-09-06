package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/audit"
)

func TestDestination(t *testing.T) {
	cases := []struct {
		name, method, raw, host, wantHost, wantPort string
	}{
		{"connect", http.MethodConnect, "", "api.example.test:8443", "api.example.test", "8443"},
		{"http default port", http.MethodGet, "http://api.example.test/path", "", "api.example.test", "80"},
		{"https default port", http.MethodGet, "https://api.example.test/path", "", "api.example.test", "443"},
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
	// 502 — the point is it got past the allowlist, not that it connected.
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
