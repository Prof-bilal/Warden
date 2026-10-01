package mcpbridge

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

func RandomID() string {
	var b [24]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}

type Factory func(context.Context, string) (*Session, error)
type httpSession struct {
	session *Session
	owner   string
	used    time.Time
}
type HTTP struct {
	ctx       context.Context
	factory   Factory
	auth      *Auth
	publicURL string
	authority string
	origins   []string
	maxBytes  int
	mu        sync.Mutex
	sessions  map[string]*httpSession
	control   string
	stop      func()
}

func NewHTTP(ctx context.Context, factory Factory, auth *Auth, publicURL string, origins []string, maxBytes int, control string, stop func()) (*HTTP, error) {
	u, e := url.Parse(publicURL)
	if e != nil || u.Host == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Path != "/mcp" || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("public URL must end in /mcp without credentials or query")
	}
	if auth.Issuer != "" && auth.Audience != publicURL {
		return nil, fmt.Errorf("OAuth audience must equal the public MCP URL")
	}
	if u.Scheme == "http" && (net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback()) {
		return nil, fmt.Errorf("non-loopback public URL requires HTTPS")
	}
	for _, origin := range origins {
		o, e := url.Parse(origin)
		if e != nil || o.Host == "" || o.User != nil || o.RawQuery != "" || o.Fragment != "" || o.Path != "" || (o.Scheme != "https" && o.Scheme != "http") {
			return nil, fmt.Errorf("origins must be exact HTTP(S) origins")
		}
	}
	h := &HTTP{ctx: ctx, factory: factory, auth: auth, publicURL: publicURL, authority: u.Host, origins: origins, maxBytes: maxBytes, sessions: map[string]*httpSession{}, control: control, stop: stop}
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				h.Close()
				return
			case <-t.C:
				h.expire()
			}
		}
	}()
	return h, nil
}
func (h *HTTP) expire() {
	h.mu.Lock()
	var closed []*Session
	for id, s := range h.sessions {
		if time.Since(s.used) > 15*time.Minute {
			delete(h.sessions, id)
			closed = append(closed, s.session)
		}
	}
	h.mu.Unlock()
	for _, s := range closed {
		_ = s.Close()
	}
}
func (h *HTTP) Close() {
	h.mu.Lock()
	all := h.sessions
	h.sessions = map[string]*httpSession{}
	h.mu.Unlock()
	for _, s := range all {
		_ = s.session.Close()
	}
}
func (h *HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Host != h.authority {
		http.Error(w, "unrecognized authority", 403)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" && !contains(h.origins, origin) {
		http.Error(w, "origin denied", 403)
		return
	}
	if r.URL.Path == "/.well-known/oauth-protected-resource/mcp" {
		if r.Method != "GET" || h.auth.Issuer == "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"resource": h.publicURL, "authorization_servers": []string{h.auth.Issuer}, "scopes_supported": []string{"mcp"}, "bearer_methods_supported": []string{"header"}})
		return
	}
	if r.URL.Path == "/__warden/stop" {
		if r.Method != "POST" || h.control == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Warden-Control")), []byte(h.control)) != 1 {
			http.Error(w, "unauthorized", 401)
			return
		}
		w.WriteHeader(202)
		_, _ = io.WriteString(w, "managed connections stopping\n")
		go h.stop()
		return
	}
	if r.URL.Path != "/mcp" {
		http.NotFound(w, r)
		return
	}
	owner, e := h.auth.Identity(r.Header.Get("Authorization"))
	if e != nil {
		if h.auth.Issuer != "" {
			u, _ := url.Parse(h.publicURL)
			u.Path = "/.well-known/oauth-protected-resource/mcp"
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+u.String()+`", scope="mcp"`)
		} else {
			w.Header().Set("WWW-Authenticate", "Bearer")
		}
		http.Error(w, "unauthorized", 401)
		return
	}
	if r.Method == "GET" {
		w.Header().Set("Allow", "POST, DELETE")
		http.Error(w, "standalone SSE streams are not supported", 405)
		return
	}
	if r.Method != "POST" && r.Method != "DELETE" {
		http.Error(w, "method unsupported", 405)
		return
	}
	id := r.Header.Get("Mcp-Session-Id")
	if version := r.Header.Get("MCP-Protocol-Version"); id != "" && version != Protocol {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write(Error(nil, -32602, "supported protocol: "+Protocol+"; 2026-07-28/MRTR unsupported"))
		return
	}
	if r.Method == "DELETE" {
		h.mu.Lock()
		entry := h.sessions[id]
		if entry == nil || entry.owner != owner {
			h.mu.Unlock()
			http.Error(w, "session not found", 404)
			return
		}
		delete(h.sessions, id)
		h.mu.Unlock()
		_ = entry.session.Close()
		w.WriteHeader(204)
		return
	}
	contentType, _, contentError := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if contentError != nil || contentType != "application/json" {
		http.Error(w, "application/json required", 415)
		return
	}
	if !strings.Contains(r.Header.Get("Accept"), "application/json") || !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		http.Error(w, "Accept header required", 400)
		return
	}
	b, e := io.ReadAll(http.MaxBytesReader(w, r.Body, int64(h.maxBytes)))
	if e != nil {
		http.Error(w, "message too large", 413)
		return
	}
	m, e := Parse(b)
	if e != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write(Error(nil, -32700, "invalid JSON-RPC"))
		return
	}
	if version := r.Header.Get("MCP-Protocol-Version"); id == "" && m.Method != "initialize" && version != "" && version != Protocol {
		http.Error(w, "unsupported protocol version", 400)
		return
	}
	h.mu.Lock()
	entry := h.sessions[id]
	if m.Method == "initialize" && id == "" {
		count := 0
		for _, s := range h.sessions {
			if s.owner == owner {
				count++
			}
		}
		if count >= 4 || len(h.sessions) >= 32 {
			h.mu.Unlock()
			http.Error(w, "session capacity reached", 429)
			return
		}
		id = RandomID()
		session, e := h.factory(h.ctx, id)
		if e != nil {
			h.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(503)
			_, _ = w.Write(Error(m.ID, -32603, "managed upstream unavailable; no fallback"))
			return
		}
		entry = &httpSession{session, owner, time.Now()}
		h.sessions[id] = entry
	} else if entry == nil || entry.owner != owner {
		h.mu.Unlock()
		http.Error(w, "session not found", 404)
		return
	}
	entry.used = time.Now()
	h.mu.Unlock()
	out := entry.session.Handle(r.Context(), b)
	if m.Method == "initialize" {
		response, _ := Parse(out)
		if len(response.Error) > 0 {
			h.mu.Lock()
			delete(h.sessions, id)
			h.mu.Unlock()
			_ = entry.session.Close()
		} else {
			w.Header().Set("Mcp-Session-Id", id)
		}
	}
	if len(out) == 0 {
		w.WriteHeader(202)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if len(m.ID) == 0 {
		w.WriteHeader(400)
	}
	_, _ = w.Write(out)
}
