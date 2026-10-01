// Package mcpbridge provides a bounded, deny-by-default MCP gateway. It is
// separate from the legacy custom TCP proxy so existing callers keep their API.
package mcpbridge

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

const Protocol = "2025-11-25"

type ToolRule struct {
	DefinitionSHA256 string `json:"definition_sha256,omitempty"`
	// Exact string values at JSON Pointer paths. Missing/non-string values deny.
	Arguments map[string][]string `json:"arguments,omitempty"`
}
type Rules struct {
	Version        int                 `json:"version"`
	Tools          map[string]ToolRule `json:"tools"`
	Resources      []string            `json:"resources"`
	Prompts        []string            `json:"prompts"`
	MaxBytes       int                 `json:"max_bytes"`
	TimeoutSeconds int                 `json:"timeout_seconds"`
}

func Hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// DefinitionHash preserves numeric lexemes, avoiding float64 rounding that
// could hide a schema change above JavaScript's safe integer range.
func DefinitionHash(raw []byte) (string, error) {
	if e := StrictJSON(raw); e != nil {
		return "", e
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var definition map[string]any
	if e := d.Decode(&definition); e != nil || definition == nil {
		return "", fmt.Errorf("tool definition must be an object")
	}
	canonical, e := json.Marshal(definition)
	if e != nil {
		return "", e
	}
	return Hash(canonical), nil
}

// StrictJSON rejects duplicate keys anywhere, trailing values and JSON batches.
func StrictJSON(b []byte) error {
	if !utf8.Valid(b) {
		return fmt.Errorf("invalid JSON UTF-8")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 64 {
			return fmt.Errorf("JSON nesting limit")
		}
		t, e := d.Token()
		if e != nil {
			return e
		}
		if delim, ok := t.(json.Delim); ok {
			switch delim {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					k, e := d.Token()
					if e != nil {
						return e
					}
					key, ok := k.(string)
					if !ok || seen[key] {
						return fmt.Errorf("duplicate JSON key")
					}
					seen[key] = true
					if e = walk(depth + 1); e != nil {
						return e
					}
				}
			case '[':
				for d.More() {
					if e = walk(depth + 1); e != nil {
						return e
					}
				}
			default:
				return fmt.Errorf("invalid JSON delimiter")
			}
			_, e = d.Token()
			return e
		}
		return nil
	}
	if e := walk(0); e != nil {
		return e
	}
	if _, e := d.Token(); e != io.EOF {
		return fmt.Errorf("trailing JSON value")
	}
	return nil
}

func LoadRules(path string) (Rules, string, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return Rules{}, "", e
	}
	if len(b) > 1024*1024 {
		return Rules{}, "", fmt.Errorf("rules exceed 1 MiB")
	}
	if e = StrictJSON(b); e != nil {
		return Rules{}, "", e
	}
	var r Rules
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(&r); e != nil {
		return r, "", e
	}
	if e = r.Validate(); e != nil {
		return r, "", e
	}
	return r, Hash(b), nil
}
func (r *Rules) Validate() error {
	if r.Version != 1 {
		return fmt.Errorf("rules version must be 1")
	}
	if r.MaxBytes == 0 {
		r.MaxBytes = 1024 * 1024
	}
	if r.TimeoutSeconds == 0 {
		r.TimeoutSeconds = 60
	}
	if r.MaxBytes < 1024 || r.MaxBytes > 8*1024*1024 || r.TimeoutSeconds < 1 || r.TimeoutSeconds > 600 {
		return fmt.Errorf("invalid gateway limits")
	}
	for name, t := range r.Tools {
		if name == "" || len(name) > 256 {
			return fmt.Errorf("invalid tool name")
		}
		if t.DefinitionSHA256 != "" {
			b, e := hex.DecodeString(t.DefinitionSHA256)
			if e != nil || len(b) != 32 {
				return fmt.Errorf("invalid tool definition digest")
			}
		}
		for p, values := range t.Arguments {
			if !strings.HasPrefix(p, "/") || len(values) == 0 {
				return fmt.Errorf("argument constraints require a JSON Pointer and exact values")
			}
			for i := 0; i < len(p); i++ {
				if p[i] == '~' {
					if i+1 >= len(p) || (p[i+1] != '0' && p[i+1] != '1') {
						return fmt.Errorf("invalid JSON Pointer escape")
					}
					i++
				}
			}
		}
	}
	return nil
}
func (r Rules) AllowCall(name string, args map[string]any) bool {
	t, ok := r.Tools[name]
	if !ok {
		return false
	}
	for pointer, allowed := range t.Arguments {
		var value any = args
		for _, part := range strings.Split(pointer[1:], "/") {
			part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
			m, ok := value.(map[string]any)
			if !ok {
				return false
			}
			value = m[part]
		}
		s, ok := value.(string)
		if !ok {
			return false
		}
		found := false
		for _, a := range allowed {
			if s == a {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

func Parse(b []byte) (Message, error) {
	var m Message
	if e := StrictJSON(b); e != nil {
		return m, e
	}
	var fields map[string]json.RawMessage
	if e := json.Unmarshal(b, &fields); e != nil {
		return m, e
	}
	if fields == nil {
		return m, fmt.Errorf("JSON-RPC object required")
	}
	if json.Unmarshal(fields["jsonrpc"], &m.JSONRPC) != nil {
		return m, fmt.Errorf("invalid jsonrpc")
	}
	if raw, ok := fields["method"]; ok {
		if json.Unmarshal(raw, &m.Method) != nil {
			return m, fmt.Errorf("invalid method")
		}
	}
	m.ID = fields["id"]
	m.Params = fields["params"]
	m.Result = fields["result"]
	m.Error = fields["error"]
	if (len(m.Result) > 0 && len(m.Error) > 0) || (m.Method != "" && (len(m.Result) > 0 || len(m.Error) > 0)) {
		return m, fmt.Errorf("ambiguous JSON-RPC envelope")
	}
	if m.JSONRPC != "2.0" {
		return m, fmt.Errorf("JSON-RPC 2.0 required")
	}
	if len(m.ID) > 0 {
		var id any
		d := json.NewDecoder(bytes.NewReader(m.ID))
		d.UseNumber()
		_ = d.Decode(&id)
		switch id.(type) {
		case string, json.Number:
		default:
			return m, fmt.Errorf("request ID must be string or number")
		}
	}
	return m, nil
}
func Encode(m Message) []byte { b, _ := json.Marshal(m); return b }
func Error(id json.RawMessage, code int, reason string) []byte {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	e, _ := json.Marshal(map[string]any{"code": code, "message": reason})
	return Encode(Message{JSONRPC: "2.0", ID: id, Error: e})
}
func Result(id json.RawMessage, value any) []byte {
	b, _ := json.Marshal(value)
	return Encode(Message{JSONRPC: "2.0", ID: id, Result: b})
}
