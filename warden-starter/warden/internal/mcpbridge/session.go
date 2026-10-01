package mcpbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

type Upstream interface {
	RoundTrip(context.Context, Message) (Message, error)
	Notify(context.Context, Message) error
	Close() error
}
type Event struct {
	Time       time.Time `json:"time"`
	Connection string    `json:"connection"`
	Method     string    `json:"method"`
	Allowed    bool      `json:"allowed"`
	Reason     string    `json:"reason"`
}

// No tool arguments, results, names, URIs, client tokens or credential values.
type Journal struct {
	mu     sync.Mutex
	Writer io.Writer
}

func (j *Journal) Record(id, method, reason string, allowed bool) error {
	if j == nil || j.Writer == nil {
		return fmt.Errorf("gateway audit writer required")
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	switch method {
	case "initialize", "ping", "tools/list", "tools/call", "resources/list", "resources/read", "prompts/list", "prompts/get", "notifications/initialized":
	default:
		method = "unsupported"
	}
	return json.NewEncoder(j.Writer).Encode(Event{time.Now().UTC(), id, method, allowed, reason})
}

type Session struct {
	mu          sync.Mutex
	upstream    Upstream
	rules       Rules
	journal     *Journal
	id          string
	initialized bool
	ready       bool
}

func NewSession(up Upstream, r Rules, j *Journal, id string) *Session {
	return &Session{upstream: up, rules: r, journal: j, id: id}
}
func (s *Session) Close() error { return s.upstream.Close() }
func (s *Session) denied(m Message, reason string) []byte {
	if e := s.journal.Record(s.id, m.Method, reason, false); e != nil {
		_ = s.Close()
		return Error(m.ID, -32603, "audit unavailable; connection closed")
	}
	return Error(m.ID, -32001, reason)
}
func (s *Session) Handle(ctx context.Context, b []byte) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, e := Parse(b)
	if e != nil {
		return s.denied(Message{}, "invalid or ambiguous JSON-RPC message")
	}
	if len(b) > s.rules.MaxBytes {
		return s.denied(m, "message exceeds policy limit")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.rules.TimeoutSeconds)*time.Second)
	defer cancel()
	if m.Method == "notifications/initialized" && len(m.ID) == 0 {
		if !s.initialized {
			return s.denied(m, "initialize first")
		}
		s.ready = true
		if e = s.upstream.Notify(ctx, m); e != nil {
			return s.denied(m, "upstream unavailable")
		}
		return nil
	}
	if len(m.ID) == 0 || m.Method == "" {
		return s.denied(m, "unsupported notification or client response")
	}
	if m.Method == "initialize" {
		if s.initialized {
			return s.denied(m, "already initialized")
		}
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
			ClientInfo      any    `json:"clientInfo"`
		}
		if json.Unmarshal(m.Params, &p) != nil || p.ProtocolVersion == "" || len(p.ProtocolVersion) > 32 {
			return s.denied(m, "initialize requires a protocol version")
		}
		// Offer our supported version when a client requests another one. The
		// client must accept this response before sending initialized; no newer
		// protocol capabilities or server requests are granted by negotiation.
		// This bounded gateway never grants sampling, elicitation, roots or tasks.
		m.Params, _ = json.Marshal(map[string]any{"protocolVersion": Protocol, "clientInfo": p.ClientInfo, "capabilities": map[string]any{}})
	} else if !s.ready {
		return s.denied(m, "complete initialization first")
	}
	var params struct {
		Name      string         `json:"name"`
		URI       string         `json:"uri"`
		Arguments map[string]any `json:"arguments"`
	}
	if len(m.Params) > 0 {
		var fields map[string]json.RawMessage
		if json.Unmarshal(m.Params, &fields) != nil || fields == nil {
			return s.denied(m, "invalid params")
		}
		if raw, ok := fields["name"]; ok && json.Unmarshal(raw, &params.Name) != nil {
			return s.denied(m, "invalid name")
		}
		if raw, ok := fields["uri"]; ok && json.Unmarshal(raw, &params.URI) != nil {
			return s.denied(m, "invalid URI")
		}
		if raw, ok := fields["arguments"]; ok && json.Unmarshal(raw, &params.Arguments) != nil {
			return s.denied(m, "invalid arguments")
		}
	}
	switch m.Method {
	case "initialize", "ping", "tools/list", "resources/list", "prompts/list":
	case "tools/call":
		if !s.rules.AllowCall(params.Name, params.Arguments) {
			return s.denied(m, "tool or arguments denied by policy")
		}
		// Refresh immediately before every call. A changed pinned definition
		// cannot reuse a previous discovery decision.
		list, e := s.upstream.RoundTrip(ctx, Message{JSONRPC: "2.0", ID: m.ID, Method: "tools/list"})
		if e != nil || len(list.Error) > 0 {
			return s.denied(m, "tool discovery unavailable")
		}
		filtered, e := s.filterList(list, "tools")
		if e != nil {
			return s.denied(m, "invalid discovery response")
		}
		var tools struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		}
		_ = json.Unmarshal(filtered.Result, &tools)
		found := false
		for _, t := range tools.Tools {
			if t.Name == params.Name {
				found = true
			}
		}
		if !found {
			return s.denied(m, "tool absent or definition changed")
		}
	case "resources/read":
		if !contains(s.rules.Resources, params.URI) {
			return s.denied(m, "resource denied by policy")
		}
	case "prompts/get":
		if !contains(s.rules.Prompts, params.Name) {
			return s.denied(m, "prompt denied by policy")
		}
	default:
		return s.denied(m, "unsupported capability or method")
	}
	if e = s.journal.Record(s.id, m.Method, "policy permits request", true); e != nil {
		_ = s.Close()
		return Error(m.ID, -32603, "audit unavailable; connection closed")
	}
	out, e := s.upstream.RoundTrip(ctx, m)
	if e != nil {
		return s.denied(m, "upstream failed or request cancelled")
	}
	if len(out.Error) > 0 {
		return Encode(out)
	}
	var result map[string]json.RawMessage
	if json.Unmarshal(out.Result, &result) != nil {
		return s.denied(m, "invalid upstream result")
	}
	if _, ok := result["inputRequests"]; ok {
		return s.denied(m, "MRTR and server input requests are unsupported")
	}
	switch m.Method {
	case "initialize":
		var version string
		_ = json.Unmarshal(result["protocolVersion"], &version)
		if version != Protocol {
			return s.denied(m, "upstream negotiated unsupported protocol")
		}
		caps := map[string]any{"tools": map[string]any{}}
		if len(s.rules.Resources) > 0 {
			caps["resources"] = map[string]any{}
		}
		if len(s.rules.Prompts) > 0 {
			caps["prompts"] = map[string]any{}
		}
		result["capabilities"], _ = json.Marshal(caps)
		out.Result, _ = json.Marshal(result)
		s.initialized = true
	case "tools/list", "resources/list", "prompts/list":
		out, e = s.filterList(out, m.Method[:len(m.Method)-5])
		if e != nil {
			return s.denied(m, "invalid discovery response")
		}
	}
	return Encode(out)
}
func contains(list []string, s string) bool {
	for _, v := range list {
		if s == v {
			return true
		}
	}
	return false
}
func (s *Session) filterList(m Message, kind string) (Message, error) {
	var result map[string]json.RawMessage
	if json.Unmarshal(m.Result, &result) != nil {
		return m, fmt.Errorf("invalid list")
	}
	var items []json.RawMessage
	if json.Unmarshal(result[kind], &items) != nil {
		return m, fmt.Errorf("invalid items")
	}
	kept := []json.RawMessage{}
	seen := map[string]bool{}
	for _, item := range items {
		var v struct {
			Name string `json:"name"`
			URI  string `json:"uri"`
		}
		if json.Unmarshal(item, &v) != nil {
			return m, fmt.Errorf("invalid item")
		}
		key := v.Name
		if kind == "resources" {
			key = v.URI
		}
		if key == "" || seen[key] {
			return m, fmt.Errorf("duplicate discovery item")
		}
		seen[key] = true
		allow := false
		switch kind {
		case "tools":
			t, ok := s.rules.Tools[v.Name]
			allow = ok
			if ok && t.DefinitionSHA256 != "" {
				digest, e := DefinitionHash(item)
				allow = e == nil && digest == t.DefinitionSHA256
			}
		case "resources":
			allow = contains(s.rules.Resources, v.URI)
		case "prompts":
			allow = contains(s.rules.Prompts, v.Name)
		}
		if allow {
			kept = append(kept, item)
		}
	}
	// Pagination is not advertised: refuse paginated discovery rather than
	// silently omit a page or accept a cursor with different permissions.
	if len(result["nextCursor"]) > 0 && string(result["nextCursor"]) != "null" {
		return m, fmt.Errorf("paginated discovery unsupported")
	}
	result[kind], _ = json.Marshal(kept)
	m.Result, _ = json.Marshal(result)
	return m, nil
}
