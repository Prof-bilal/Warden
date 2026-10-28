// Package agentworkspace creates sanitized copies and validates changes before
// any host write. Secret discovery is exclusively by explicit field selectors.
package agentworkspace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type document struct {
	format string
	tree   any
	lines  []string
	keys   map[string]int
}

func parse(data []byte, format string) (*document, error) {
	d := &document{format: format}
	switch format {
	case "dotenv":
		d.lines = strings.Split(string(data), "\n")
		d.keys = map[string]int{}
		for i, line := range d.lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			k, v, ok := strings.Cut(strings.TrimPrefix(trimmed, "export "), "=")
			k = strings.TrimSpace(k)
			if !ok || k == "" || strings.ContainsAny(k, " \t\r") {
				return nil, fmt.Errorf("unsupported dotenv syntax")
			}
			if _, ok := d.keys[k]; ok {
				return nil, fmt.Errorf("duplicate dotenv key")
			}
			// Multiline values need a source-aware parser; never guess their boundary.
			v = strings.TrimSpace(v)
			if strings.HasPrefix(v, "\"") || strings.HasPrefix(v, "'") {
				quote := v[0]
				end := -1
				for j := 1; j < len(v); j++ {
					if v[j] == '\\' && quote == '"' {
						j++
						continue
					}
					if v[j] == quote {
						end = j
						break
					}
				}
				if end < 0 || (strings.TrimSpace(v[end+1:]) != "" && !strings.HasPrefix(strings.TrimSpace(v[end+1:]), "#")) {
					return nil, fmt.Errorf("unsupported multiline dotenv value")
				}
			}
			d.keys[k] = i
		}
	case "json":
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		tree, err := jsonValue(dec)
		if err != nil {
			return nil, fmt.Errorf("invalid or duplicate-key JSON")
		}
		if _, err = dec.Token(); err != io.EOF {
			return nil, fmt.Errorf("trailing JSON content")
		}
		d.tree = tree
	case "yaml":
		var root yaml.Node
		dec := yaml.NewDecoder(bytes.NewReader(data))
		if err := dec.Decode(&root); err != nil {
			return nil, fmt.Errorf("invalid YAML")
		}
		var extra yaml.Node
		if dec.Decode(&extra) != io.EOF {
			return nil, fmt.Errorf("multiple YAML documents are unsupported")
		}
		if err := safeYAML(&root); err != nil {
			return nil, err
		}
		if err := root.Decode(&d.tree); err != nil {
			return nil, fmt.Errorf("invalid YAML mapping")
		}
	case "toml":
		var tree map[string]any
		if err := toml.Unmarshal(data, &tree); err != nil {
			return nil, fmt.Errorf("invalid or duplicate-key TOML")
		}
		d.tree = tree
	default:
		return nil, fmt.Errorf("unsupported format")
	}
	return d, nil
}

func jsonValue(dec *json.Decoder) (any, error) {
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := t.(json.Delim); ok {
		switch delim {
		case '{':
			m := map[string]any{}
			for dec.More() {
				k, e := dec.Token()
				if e != nil {
					return nil, e
				}
				key, ok := k.(string)
				if !ok {
					return nil, fmt.Errorf("invalid key")
				}
				if _, ok = m[key]; ok {
					return nil, fmt.Errorf("duplicate key")
				}
				v, e := jsonValue(dec)
				if e != nil {
					return nil, e
				}
				m[key] = v
			}
			_, err = dec.Token()
			return m, err
		case '[':
			a := make([]any, 0)
			for dec.More() {
				v, e := jsonValue(dec)
				if e != nil {
					return nil, e
				}
				a = append(a, v)
			}
			_, err = dec.Token()
			return a, err
		}
		return nil, fmt.Errorf("unexpected delimiter")
	}
	return t, nil
}

func safeYAML(n *yaml.Node) error {
	if n.Kind == yaml.AliasNode || n.Anchor != "" || n.Tag == "!!merge" {
		return fmt.Errorf("YAML anchors, aliases and merge keys are unsupported")
	}
	if n.Kind == yaml.ScalarNode {
		switch n.Tag {
		case "!!str", "!!bool", "!!null", "!!int", "!!float", "!!timestamp":
		default:
			return fmt.Errorf("custom YAML tags are unsupported")
		}
	}
	if n.Kind == yaml.MappingNode {
		seen := map[string]bool{}
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Tag != "!!str" || seen[k.Value] {
				return fmt.Errorf("duplicate or non-string YAML key")
			}
			seen[k.Value] = true
		}
	}
	for _, c := range n.Content {
		if err := safeYAML(c); err != nil {
			return err
		}
	}
	return nil
}

func (d *document) replace(field string, replacement *string) (string, error) {
	if d.format == "dotenv" {
		i, ok := d.keys[field]
		if !ok {
			return "", fmt.Errorf("protected key missing")
		}
		prefix, value, _ := strings.Cut(d.lines[i], "=")
		if replacement != nil {
			d.lines[i] = prefix + "=" + *replacement
		}
		return value, nil
	}
	parts := strings.Split(strings.TrimPrefix(field, "/"), "/")
	current := d.tree
	for i, p := range parts {
		// JSON Pointer escaping; arrays are deliberately not selectable in v1.
		for j := 0; j < len(p); j++ {
			if p[j] == '~' {
				if j+1 == len(p) || (p[j+1] != '0' && p[j+1] != '1') {
					return "", fmt.Errorf("invalid JSON Pointer escape")
				}
				j++
			}
		}
		p = strings.ReplaceAll(strings.ReplaceAll(p, "~1", "/"), "~0", "~")
		m, ok := current.(map[string]any)
		if !ok {
			return "", fmt.Errorf("protected path must traverse object/table fields")
		}
		v, ok := m[p]
		if !ok {
			return "", fmt.Errorf("protected field missing")
		}
		if i == len(parts)-1 {
			s, ok := v.(string)
			if !ok {
				return "", fmt.Errorf("protected value must be a string")
			}
			if replacement != nil {
				m[p] = *replacement
			}
			return s, nil
		}
		current = v
	}
	return "", fmt.Errorf("invalid field")
}

func (d *document) bytes() ([]byte, error) {
	switch d.format {
	case "dotenv":
		return []byte(strings.Join(d.lines, "\n")), nil
	case "json":
		b, e := json.MarshalIndent(d.tree, "", "  ")
		return append(b, '\n'), e
	case "yaml":
		return yaml.Marshal(d.tree)
	case "toml":
		return toml.Marshal(d.tree)
	}
	return nil, fmt.Errorf("unsupported format")
}
