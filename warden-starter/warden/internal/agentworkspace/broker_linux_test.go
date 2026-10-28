//go:build linux

package agentworkspace

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/policy"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestBrokerFixedOperationRedactsEchoAndAuditsNames(t *testing.T) {
	secret := "credential-test-123"
	var logs bytes.Buffer
	calls := 0
	client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://api.example.com/status" || r.Method != "GET" || r.Header.Get("Authorization") != "Bearer "+secret {
			t.Fatal("credential or fixed request incorrect")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Set-Cookie": []string{secret}}, Body: io.NopCloser(strings.NewReader(secret + " " + base64.StdEncoding.EncodeToString([]byte(secret))))}, nil
	})}
	h := brokerHandler(map[string]policy.SecretProfile{"status": {URL: "https://api.example.com/status", Env: "TOKEN"}}, map[string]string{"status": secret}, audit.New(&logs), client)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest("GET", "/profiles/status", nil))
	if calls != 1 || res.Code != 200 || strings.Contains(res.Body.String(), secret) || res.Header().Get("Set-Cookie") != "" {
		t.Fatal("unsafe broker response")
	}
	if !strings.Contains(res.Body.String(), "[WARDEN_REDACTED]") || strings.Contains(logs.String(), secret) {
		t.Fatal("echo/log redaction failed")
	}
	for _, r := range []*http.Request{
		httptest.NewRequest("POST", "/profiles/status", strings.NewReader("arbitrary")),
		httptest.NewRequest("GET", "/profiles/status?url=https://other.example.com", nil),
		httptest.NewRequest("GET", "/profiles/missing", nil),
	} {
		res := httptest.NewRecorder()
		h.ServeHTTP(res, r)
		if res.Code != 403 {
			t.Fatal("request override accepted")
		}
	}
	if calls != 1 {
		t.Fatal("rejected request reached upstream")
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, errors.New("audit failed") }
func TestBrokerFailsClosedWithoutAudit(t *testing.T) {
	client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		t.Fatal("upstream reached without audit")
		return nil, nil
	})}
	h := brokerHandler(map[string]policy.SecretProfile{"status": {URL: "https://api.example.com/status"}}, map[string]string{"status": "secret"}, audit.New(failedWriter{}), client)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest("GET", "/profiles/status", nil))
	if res.Code != 500 {
		t.Fatal("audit failure ignored")
	}
}
