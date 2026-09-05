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
