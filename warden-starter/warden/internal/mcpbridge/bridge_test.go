package mcpbridge

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeUp struct {
	calls       []string
	initial     json.RawMessage
	description string
	closed      bool
}

func (f *fakeUp) Close() error                          { f.closed = true; return nil }
func (f *fakeUp) Notify(context.Context, Message) error { return nil }
func (f *fakeUp) RoundTrip(ctx context.Context, m Message) (Message, error) {
	f.calls = append(f.calls, m.Method)
	var value any = map[string]any{}
	switch m.Method {
	case "initialize":
		f.initial = m.Params
		value = map[string]any{"protocolVersion": Protocol, "capabilities": map[string]any{"tools": map[string]any{"listChanged": true}, "resources": map[string]any{"subscribe": true}}}
	case "tools/list":
		value = map[string]any{"tools": []any{map[string]any{"name": "read", "description": f.description, "inputSchema": map[string]any{"type": "object"}}, map[string]any{"name": "write"}}}
	case "tools/call":
		value = map[string]any{"content": []any{map[string]any{"type": "text", "text": "allowed fixture"}}}
	case "resources/list":
		value = map[string]any{"resources": []any{map[string]any{"uri": "file:///allowed"}, map[string]any{"uri": "file:///secret"}}}
	case "prompts/list":
		value = map[string]any{"prompts": []any{map[string]any{"name": "allowed"}, map[string]any{"name": "secret"}}}
	}
	m.Method = ""
	m.Params = nil
	m.Result, _ = json.Marshal(value)
	return m, nil
}
func TestInitializeNegotiatesSupportedSubsetWithoutClientCapabilities(t *testing.T) {
	up := &fakeUp{}
	r := Rules{Version: 1, Tools: map[string]ToolRule{"read": {}}}
	_ = r.Validate()
	var journal bytes.Buffer
	s := NewSession(up, r, &Journal{Writer: &journal}, "negotiation")
	response := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2026-07-28","capabilities":{"sampling":{},"tasks":{},"elicitation":{},"roots":{}},"clientInfo":{"name":"new-client","version":"1"}}}`))
	m, e := Parse(response)
	if e != nil || len(m.Error) != 0 || !strings.Contains(string(m.Result), Protocol) {
		t.Fatalf("version negotiation failed: %s", response)
	}
	if !strings.Contains(string(up.initial), Protocol) || strings.Contains(string(up.initial), "sampling") || strings.Contains(string(up.initial), "elicitation") || strings.Contains(string(up.initial), "roots") || strings.Contains(string(up.initial), "tasks") {
		t.Fatalf("capabilities widened upstream: %s", up.initial)
	}
	s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	before := len(up.calls)
	denied := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":2,"method":"tasks/list"}`))
	if !strings.Contains(string(denied), "unsupported") || len(up.calls) != before {
		t.Fatal("unsupported capability forwarded after negotiation")
	}
}
func ready(t *testing.T, s *Session) {
	t.Helper()
	out := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{"sampling":{}},"clientInfo":{"name":"test","version":"1"}}}`))
	m, e := Parse(out)
	if e != nil || len(m.Error) > 0 {
		t.Fatalf("initialize: %s", out)
	}
	if strings.Contains(string(m.Result), "subscribe") || strings.Contains(string(m.Result), "listChanged") {
		t.Fatal("unsupported capabilities advertised")
	}
	s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
}
func TestFilteringAndDefinitionDrift(t *testing.T) {
	up := &fakeUp{description: "reviewed"}
	definition, _ := json.Marshal(map[string]any{"name": "read", "description": "reviewed", "inputSchema": map[string]any{"type": "object"}})
	r := Rules{Version: 1, Tools: map[string]ToolRule{"read": {DefinitionSHA256: Hash(definition), Arguments: map[string][]string{"/path": {"/selected/hello"}}}}, Resources: []string{"file:///allowed"}, Prompts: []string{"allowed"}}
	if e := r.Validate(); e != nil {
		t.Fatal(e)
	}
	var log bytes.Buffer
	s := NewSession(up, r, &Journal{Writer: &log}, "c")
	ready(t, s)
	for _, request := range []string{`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`, `{"jsonrpc":"2.0","id":4,"method":"resources/list"}`, `{"jsonrpc":"2.0","id":5,"method":"prompts/list"}`} {
		out := s.Handle(context.Background(), []byte(request))
		if strings.Contains(string(out), "secret") || strings.Contains(string(out), `"write"`) {
			t.Fatal("discovery leaked disallowed entries")
		}
	}
	allowed := []byte(`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"read","arguments":{"path":"/selected/hello"}}}`)
	if out := s.Handle(context.Background(), allowed); !bytes.Contains(out, []byte("allowed fixture")) {
		t.Fatalf("allowed: %s", out)
	}
	before := len(up.calls)
	mixed := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"write","Name":"read","arguments":{"path":"/selected/hello"}}}`))
	if !bytes.Contains(mixed, []byte(`"code":-32001`)) {
		t.Fatal("case alias bypassed tool policy")
	}
	for _, request := range []string{`{"jsonrpc":"2.0","id":"denied","method":"tools/call","params":{"name":"write","arguments":{"token":"super-secret"}}}`, `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"read","arguments":{"path":"/outside"}}}`, `{"jsonrpc":"2.0","id":9,"method":"resources/read","params":{"uri":"file:///secret"}}`, `{"jsonrpc":"2.0","id":10,"method":"tasks/get"}`, `{"jsonrpc":"2.0","id":11,"method":"tools/call","params":{"name":"read","name":"write"}}`} {
		out := s.Handle(context.Background(), []byte(request))
		if !bytes.Contains(out, []byte(`"code":-32001`)) {
			t.Fatalf("not denied: %s", out)
		}
	}
	if len(up.calls) != before {
		t.Fatal("denied request reached upstream")
	}
	if strings.Contains(log.String(), "super-secret") || strings.Contains(log.String(), "/outside") {
		t.Fatal("audit leaked payload")
	}
	up.description = "changed"
	out := s.Handle(context.Background(), allowed)
	if !bytes.Contains(out, []byte("definition changed")) {
		t.Fatalf("changed schema reached call: %s", out)
	}
}
func TestStrictJSONAndFailClosedAudit(t *testing.T) {
	a, e := DefinitionHash([]byte(`{"name":"read","inputSchema":{"maximum":9007199254740992}}`))
	if e != nil {
		t.Fatal(e)
	}
	b, e := DefinitionHash([]byte(`{"name":"read","inputSchema":{"maximum":9007199254740993}}`))
	if e != nil || a == b {
		t.Fatal("numeric schema drift hidden by rounding")
	}
	for _, raw := range []string{`{"x":1,"x":2}`, `{"a":{"x":1,"x":2}}`, `{} {}`, `{"x":[1,]}`} {
		if StrictJSON([]byte(raw)) == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	up := &fakeUp{}
	r := Rules{Version: 1}
	_ = r.Validate()
	s := NewSession(up, r, nil, "c")
	out := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call"}`))
	if !up.closed || !bytes.Contains(out, []byte("audit unavailable")) {
		t.Fatal("audit failure did not close connection")
	}
}

func oauthFixture(t *testing.T) (*Auth, func(string, string, int64) string) {
	t.Helper()
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	enc := base64.RawURLEncoding.EncodeToString
	raw, _ := json.Marshal(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "one", "alg": "RS256", "n": enc(key.N.Bytes()), "e": enc(big.NewInt(int64(key.E)).Bytes())}}})
	file := filepath.Join(t.TempDir(), "jwks.json")
	_ = os.WriteFile(file, raw, 0600)
	a, e := LoadAuth("", "https://issuer.example", "https://gateway.example/mcp", file)
	if e != nil {
		t.Fatal(e)
	}
	return a, func(subject, audience string, exp int64) string {
		h, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "one"})
		p, _ := json.Marshal(map[string]any{"iss": a.Issuer, "sub": subject, "aud": audience, "exp": exp, "scope": "mcp"})
		unsigned := enc(h) + "." + enc(p)
		sum := sha256.Sum256([]byte(unsigned))
		sig, _ := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
		return unsigned + "." + enc(sig)
	}
}
func TestOAuthClaimsAndSessionIsolation(t *testing.T) {
	a, token := oauthFixture(t)
	good := token("alice", a.Audience, time.Now().Add(time.Hour).Unix())
	badAudience := token("alice", "https://other.example/mcp", time.Now().Add(time.Hour).Unix())
	expired := token("alice", a.Audience, time.Now().Add(-time.Hour).Unix())
	for _, v := range []string{badAudience, expired, good + "tampered"} {
		if _, e := a.Identity("Bearer " + v); e == nil {
			t.Fatal("invalid OAuth token accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := Rules{Version: 1, Tools: map[string]ToolRule{"read": {}}}
	_ = r.Validate()
	var log bytes.Buffer
	h, e := NewHTTP(ctx, func(ctx context.Context, id string) (*Session, error) {
		return NewSession(&fakeUp{}, r, &Journal{Writer: &log}, id), nil
	}, a, a.Audience, nil, r.MaxBytes, "", func() {})
	if e != nil {
		t.Fatal(e)
	}
	defer h.Close()
	request := func(method, session, bearer, origin, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, a.Audience, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+bearer)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("MCP-Protocol-Version", Protocol)
		req.Header.Set("Mcp-Session-Id", session)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	init := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`
	w := request("POST", "", good, "", init)
	if w.Code != 200 || w.Header().Get("Mcp-Session-Id") == "" {
		t.Fatalf("HTTP init: %d %s", w.Code, w.Body.String())
	}
	id := w.Header().Get("Mcp-Session-Id")
	bob := token("bob", a.Audience, time.Now().Add(time.Hour).Unix())
	if w = request("POST", id, bob, "", `{"jsonrpc":"2.0","id":2,"method":"ping"}`); w.Code != 404 {
		t.Fatal("cross-principal session accessible")
	}
	if w = request("POST", id, good, "https://evil.example", init); w.Code != 403 {
		t.Fatal("bad origin accepted")
	}
	if w = request("POST", id, good, "", `{"jsonrpc":"2.0","method":"notifications/initialized"}`); w.Code != 202 {
		t.Fatal("notification not accepted")
	}
	if w = request("POST", id, good, "", `{"jsonrpc":"2.0","id":"blocked","method":"tools/call","params":{"name":"write"}}`); !strings.Contains(w.Body.String(), `"id":"blocked"`) {
		t.Fatal("denial lost correlation")
	}
	if w = request("DELETE", id, good, "", ""); w.Code != 204 {
		t.Fatal("session cleanup failed")
	}
}
func TestHTTPNegotiationRequiresChosenVersionOnSessionRequests(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rules := Rules{Version: 1, Tools: map[string]ToolRule{"read": {}}}
	if err := rules.Validate(); err != nil {
		t.Fatal(err)
	}
	var journal bytes.Buffer
	auth := &Auth{Token: strings.Repeat("x", 32)}
	up := &fakeUp{}
	h, err := NewHTTP(ctx, func(context.Context, string) (*Session, error) {
		return NewSession(up, rules, &Journal{Writer: &journal}, "negotiation"), nil
	}, auth, "http://127.0.0.1:8788/mcp", nil, rules.MaxBytes, "", func() {})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	request := func(session, version, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "http://127.0.0.1:8788/mcp", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+auth.Token)
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Mcp-Session-Id", session)
		req.Header.Set("MCP-Protocol-Version", version)
		out := httptest.NewRecorder()
		h.ServeHTTP(out, req)
		return out
	}
	init := request("", "2026-07-28", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2026-07-28"}}`)
	id := init.Header().Get("Mcp-Session-Id")
	if init.Code != 200 || id == "" || !strings.Contains(init.Body.String(), Protocol) {
		t.Fatalf("negotiation failed: %d %s", init.Code, init.Body.String())
	}
	for _, version := range []string{"", "2026-07-28"} {
		before := len(up.calls)
		out := request(id, version, `{"jsonrpc":"2.0","id":2,"method":"ping"}`)
		if out.Code != 400 || len(up.calls) != before {
			t.Fatalf("unsupported session header reached upstream: %q, %d", version, out.Code)
		}
	}
	if out := request(id, Protocol, `{"jsonrpc":"2.0","method":"notifications/initialized"}`); out.Code != 202 {
		t.Fatalf("supported session notification failed: %d %s", out.Code, out.Body.String())
	}
	if out := request(id, Protocol, `{"jsonrpc":"2.0","id":3,"method":"ping"}`); out.Code != 200 || !strings.Contains(out.Body.String(), `"result"`) {
		t.Fatalf("supported session request failed: %d %s", out.Code, out.Body.String())
	}
}

func TestSSRFRedirectsAndRemoteSSE(t *testing.T) {
	for _, addr := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.64.0.1", "::1", "fd00::1"} {
		if PublicAddress(net.ParseIP(addr)) {
			t.Fatalf("private address allowed: %s", addr)
		}
	}
	for _, endpoint := range []string{"http://example.org/mcp", "https://user:password@example.org/mcp", "https://example.org/mcp?token=secret"} {
		if _, e := NewRemote(endpoint, "", false, 1024); e == nil {
			t.Fatal("invalid endpoint accepted")
		}
	}
	var auth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/mcp", 302)
			return
		}
		if r.Method != "POST" {
			t.Error("unexpected method")
		}
		b, _ := io.ReadAll(r.Body)
		m, _ := Parse(b)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\ndata: " + string(Result(m.ID, map[string]any{})) + "\n\n"))
	}))
	defer server.Close()
	remote, e := NewRemote(server.URL+"/mcp", "upstream-only", true, 1024)
	if e != nil {
		t.Fatal(e)
	}
	_, e = remote.RoundTrip(context.Background(), Message{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "ping"})
	if e != nil || auth != "Bearer upstream-only" {
		t.Fatalf("remote SSE failed: %v", e)
	}
	blocked, _ := NewRemote(server.URL+"/redirect", "", true, 1024)
	if _, e = blocked.RoundTrip(context.Background(), Message{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "ping"}); e == nil {
		t.Fatal("redirect followed")
	}
}
