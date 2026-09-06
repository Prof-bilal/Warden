package gateway

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Wrap produces a copy of the source gateway config where every stdio
// server's spawn command is prefixed with Warden (`warden run --policy ...`).
// Remote entries are passed through untouched.
//
// The wrapped config keeps the original env VALUE maps: at runtime the
// gateway spawns `warden run` with those values in its environment, and
// Warden forwards only the names in the policy's env.allow to the sandboxed
// server. Policy files never store secrets.
func Wrap(cfg *Config, policiesDir, wardenBin, backend string) ([]byte, error) {
	if wardenBin == "" {
		return nil, fmt.Errorf("warden binary path must not be empty")
	}
	if policiesDir == "" {
		return nil, fmt.Errorf("policies dir must not be empty")
	}
	switch cfg.Format {
	case "json":
		return wrapJSON(cfg, policiesDir, wardenBin, backend)
	default:
		return wrapYAML(cfg, policiesDir, wardenBin, backend)
	}
}

func wrapJSON(cfg *Config, policiesDir, wardenBin, backend string) ([]byte, error) {
	data, err := readRaw(cfg.Path)
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse gateway config %q: %w", cfg.Path, err)
	}
	for _, key := range []string{"mcpServers", "servers"} {
		blob, ok := raw[key]
		if !ok {
			continue
		}
		var servers map[string]map[string]any
		if err := json.Unmarshal(blob, &servers); err != nil {
			return nil, fmt.Errorf("parse %q section: %w", key, err)
		}
		for _, e := range cfg.Servers {
			m, ok := servers[e.Name]
			if !ok || e.Remote {
				continue
			}
			wrapped := WrappedCommand(wardenBin, PolicyPath(policiesDir, e.Name), backend, e)
			m["command"] = wrapped[0]
			args := make([]any, 0, len(wrapped)-1)
			for _, a := range wrapped[1:] {
				args = append(args, a)
			}
			m["args"] = args
			servers[e.Name] = m
		}
		blob, err := json.MarshalIndent(servers, "", "  ")
		if err != nil {
			return nil, err
		}
		raw[key] = blob
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func wrapYAML(cfg *Config, policiesDir, wardenBin, backend string) ([]byte, error) {
	data, err := readRaw(cfg.Path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse gateway config %q: %w", cfg.Path, err)
	}
	byName := make(map[string]ServerEntry, len(cfg.Servers))
	for _, e := range cfg.Servers {
		byName[e.Name] = e
	}
	pol := func(name string) string { return PolicyPath(policiesDir, name) }
	if ups, ok := raw["upstreams"].([]any); ok {
		for i, u := range ups {
			m, ok := u.(map[string]any)
			if !ok {
				continue
			}
			name, _ := m["id"].(string)
			if name == "" {
				name, _ = m["name"].(string)
			}
			e, ok := byName[name]
			if !ok || e.Remote {
				continue
			}
			wrapped := WrappedCommand(wardenBin, pol(name), backend, e)
			m["command"] = wrapped[0]
			args := make([]any, 0, len(wrapped)-1)
			for _, a := range wrapped[1:] {
				args = append(args, a)
			}
			m["args"] = args
			m["transport"] = "stdio"
			delete(m, "endpoint")
			delete(m, "url")
			ups[i] = m
		}
		raw["upstreams"] = ups
	}
	if bes, ok := raw["backends"].(map[string]any); ok {
		for name, b := range bes {
			m, ok := b.(map[string]any)
			if !ok {
				continue
			}
			e, ok := byName[name]
			if !ok || e.Remote {
				continue
			}
			wrapped := WrappedCommand(wardenBin, pol(name), backend, e)
			// Mikko-style backends take a single command string.
			m["command"] = shellJoin(wrapped)
			delete(m, "args")
			delete(m, "http_url")
			delete(m, "url")
			bes[name] = m
		}
		raw["backends"] = bes
	}
	// Claude-shape-in-YAML: wrap mcpServers/servers maps the same as JSON.
	for _, key := range []string{"mcpServers", "servers"} {
		sec, ok := raw[key].(map[string]any)
		if !ok {
			continue
		}
		for name, v := range sec {
			m, ok := v.(map[string]any)
			if !ok {
				continue
			}
			e, ok := byName[name]
			if !ok || e.Remote {
				continue
			}
			wrapped := WrappedCommand(wardenBin, pol(name), backend, e)
			m["command"] = wrapped[0]
			args := make([]any, 0, len(wrapped)-1)
			for _, a := range wrapped[1:] {
				args = append(args, a)
			}
			m["args"] = args
			sec[name] = m
		}
		raw[key] = sec
	}
	var sb strings.Builder
	enc := yaml.NewEncoder(&sb)
	enc.SetIndent(2)
	if err := enc.Encode(raw); err != nil {
		return nil, err
	}
	enc.Close()
	return []byte(sb.String()), nil
}

func shellJoin(argv []string) string {
	q := make([]string, len(argv))
	for i, a := range argv {
		if a == "" || strings.ContainsAny(a, " \t\n\"'\\") {
			q[i] = "'" + strings.ReplaceAll(a, "'", "'\"'\"'") + "'"
		} else {
			q[i] = a
		}
	}
	return strings.Join(q, " ")
}
