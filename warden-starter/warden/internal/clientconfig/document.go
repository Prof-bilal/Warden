package clientconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/unstable"
)

type span struct{ start, end int }
type edit struct {
	span
	text []byte
}
type node struct {
	span
	keyStart int
	members  map[string]*node
}
type Launcher struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	HadArgs bool     `json:"had_args"`
}
type Document struct {
	data    []byte
	format  string
	root    *node
	entries map[string]Launcher
	fields  map[string]map[string]span
	insert  map[string]int
}

// Parse validates the whole document, rejects duplicate JSON keys, and keeps
// byte offsets so unrelated fields, comments and interpolation remain intact.
func Parse(data []byte, format string) (*Document, error) {
	d := &Document{data: data, format: format, entries: map[string]Launcher{}, fields: map[string]map[string]span{}, insert: map[string]int{}}
	if len(data) > 8*1024*1024 {
		return nil, fmt.Errorf("configuration exceeds 8 MiB")
	}
	if format == "toml" {
		return d, d.parseTOML()
	}
	clean := append([]byte(nil), data...)
	if format == "jsonc" {
		if err := cleanJSONC(clean); err != nil {
			return nil, err
		}
	}
	if !json.Valid(clean) {
		return nil, fmt.Errorf("invalid configuration JSON")
	}
	dec := json.NewDecoder(bytes.NewReader(clean))
	dec.UseNumber()
	root, err := readNode(dec, clean)
	if err != nil {
		return nil, err
	}
	d.root = root
	section := root.members["mcpServers"]
	if section == nil {
		section = root.members["servers"]
	}
	if section == nil || section.members == nil {
		return nil, fmt.Errorf("expected mcpServers or servers object")
	}
	// Keep just the server section as the root for patching.
	d.root = section
	for name, n := range section.members {
		if n.members == nil {
			return nil, fmt.Errorf("server entry must be an object")
		}
		if n.members["url"] != nil || n.members["httpUrl"] != nil || n.members["endpoint"] != nil {
			continue
		}
		if t := n.members["type"]; t != nil {
			var typ string
			_ = json.Unmarshal(clean[t.start:t.end], &typ)
			if typ != "" && typ != "stdio" {
				continue
			}
		}
		cmd := n.members["command"]
		if cmd == nil {
			continue
		}
		var l Launcher
		if err := json.Unmarshal(clean[cmd.start:cmd.end], &l.Command); err != nil || l.Command == "" {
			return nil, fmt.Errorf("invalid server command")
		}
		if a := n.members["args"]; a != nil {
			l.HadArgs = true
			if err := json.Unmarshal(clean[a.start:a.end], &l.Args); err != nil || l.Args == nil {
				return nil, fmt.Errorf("args must be an array of strings")
			}
		}
		if l.Args == nil {
			l.Args = []string{}
		}
		d.entries[name] = l
	}
	return d, nil
}

func readNode(dec *json.Decoder, data []byte) (*node, error) {
	start := int(dec.InputOffset())
	for start < len(data) && strings.ContainsRune(" \r\n\t,:", rune(data[start])) {
		start++
	}
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	n := &node{span: span{start: start}, keyStart: start}
	if delimiter, ok := tok.(json.Delim); ok {
		if delimiter == '{' {
			n.members = map[string]*node{}
			for dec.More() {
				kstart := int(dec.InputOffset())
				for kstart < len(data) && strings.ContainsRune(" \r\n\t,", rune(data[kstart])) {
					kstart++
				}
				key, err := dec.Token()
				if err != nil {
					return nil, err
				}
				k := key.(string)
				if _, exists := n.members[k]; exists {
					return nil, fmt.Errorf("duplicate configuration key")
				}
				child, err := readNode(dec, data)
				if err != nil {
					return nil, err
				}
				child.keyStart = kstart
				n.members[k] = child
			}
		} else {
			for dec.More() {
				if _, err := readNode(dec, data); err != nil {
					return nil, err
				}
			}
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
	}
	n.end = int(dec.InputOffset())
	return n, nil
}

func cleanJSONC(b []byte) error {
	inString, escaped := false, false
	for i := 0; i < len(b); i++ {
		if inString {
			if escaped {
				escaped = false
			} else if b[i] == '\\' {
				escaped = true
			} else if b[i] == '"' {
				inString = false
			}
			continue
		}
		if b[i] == '"' {
			inString = true
			continue
		}
		if b[i] == '/' && i+1 < len(b) {
			if b[i+1] == '/' {
				for ; i < len(b) && b[i] != '\n'; i++ {
					b[i] = ' '
				}
				i--
				continue
			}
			if b[i+1] == '*' {
				start := i
				i += 2
				for i+1 < len(b) && !(b[i] == '*' && b[i+1] == '/') {
					i++
				}
				if i+1 >= len(b) {
					return fmt.Errorf("unterminated JSON comment")
				}
				i++
				for j := start; j <= i; j++ {
					if b[j] != '\n' && b[j] != '\r' {
						b[j] = ' '
					}
				}
				continue
			}
		}
	}
	inString, escaped = false, false
	for i := 0; i < len(b); i++ {
		if inString {
			if escaped {
				escaped = false
			} else if b[i] == '\\' {
				escaped = true
			} else if b[i] == '"' {
				inString = false
			}
			continue
		}
		if b[i] == '"' {
			inString = true
			continue
		}
		if b[i] == ',' {
			j := i + 1
			for j < len(b) && strings.ContainsRune(" \r\n\t", rune(b[j])) {
				j++
			}
			if j < len(b) && (b[j] == '}' || b[j] == ']') {
				b[i] = ' '
			}
		}
	}
	return nil
}

func (d *Document) Launcher(name string) (Launcher, error) {
	l, ok := d.entries[name]
	if !ok {
		return Launcher{}, fmt.Errorf("selected server is missing, remote, or has no stdio launcher")
	}
	return l, nil
}
func (d *Document) Names() []string {
	names := []string{}
	for n := range d.entries {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (d *Document) Patch(name string, l Launcher) ([]byte, error) {
	if _, err := d.Launcher(name); err != nil {
		return nil, err
	}
	edits := []edit{}
	encodedCmd, _ := json.Marshal(l.Command)
	encodedArgs, _ := json.Marshal(l.Args)
	if l.Args == nil {
		encodedArgs = []byte("[]")
	}
	if d.format == "toml" {
		fields := d.fields[name]
		if fields["command"].end == 0 {
			return nil, fmt.Errorf("only explicit [mcp_servers.NAME] tables are editable; export a launcher for other TOML layouts")
		}
		edits = append(edits, edit{fields["command"], encodedCmd})
		if a, ok := fields["args"]; ok {
			if l.HadArgs {
				edits = append(edits, edit{a, encodedArgs})
			} else {
				start := a.start
				for start > 0 && d.data[start-1] != '\n' {
					start--
				}
				end := a.end
				for end < len(d.data) && d.data[end] != '\n' {
					end++
				}
				if end < len(d.data) {
					end++
				}
				edits = append(edits, edit{span{start, end}, nil})
			}
		} else if l.HadArgs {
			at := d.insert[name]
			prefix := ""
			if at == len(d.data) && (at == 0 || d.data[at-1] != '\n') {
				prefix = "\n"
			}
			edits = append(edits, edit{span{at, at}, []byte(prefix + "args = " + string(encodedArgs) + "\n")})
		}
	} else {
		n := d.root.members[name]
		edits = append(edits, edit{n.members["command"].span, encodedCmd})
		if a := n.members["args"]; a != nil {
			if l.HadArgs {
				edits = append(edits, edit{a.span, encodedArgs})
			} else {
				clean := append([]byte(nil), d.data...)
				if d.format == "jsonc" {
					_ = cleanJSONC(clean)
				}
				start := a.keyStart
				prev := start - 1
				for prev > n.start && strings.ContainsRune(" \r\n\t", rune(clean[prev])) {
					prev--
				}
				end := a.end
				if clean[prev] == ',' {
					start = prev
				} else {
					for end < n.end && strings.ContainsRune(" \r\n\t", rune(clean[end])) {
						end++
					}
					if end < n.end && clean[end] == ',' {
						end++
					}
				}
				edits = append(edits, edit{span{start, end}, nil})
			}
		} else if l.HadArgs {
			// Insert after command, before any trailing comment/comma.
			at := n.members["command"].end
			edits = append(edits, edit{span{at, at}, []byte(", \"args\": " + string(encodedArgs))})
		}
	}
	out := applyEdits(d.data, edits)
	parsed, err := Parse(out, d.format)
	if err != nil {
		return nil, fmt.Errorf("patched configuration failed validation")
	}
	got, err := parsed.Launcher(name)
	if err != nil || !Equal(got, l) {
		return nil, fmt.Errorf("patched launcher failed validation")
	}
	return out, nil
}

func Equal(a, b Launcher) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}
func applyEdits(data []byte, edits []edit) []byte {
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	out := append([]byte(nil), data...)
	for _, e := range edits {
		next := make([]byte, 0, len(out)+len(e.text))
		next = append(next, out[:e.start]...)
		next = append(next, e.text...)
		next = append(next, out[e.end:]...)
		out = next
	}
	return out
}

func (d *Document) parseTOML() error {
	var root map[string]any
	if err := toml.Unmarshal(d.data, &root); err != nil {
		return fmt.Errorf("invalid configuration TOML")
	}
	servers, ok := root["mcp_servers"].(map[string]any)
	if !ok {
		return fmt.Errorf("expected mcp_servers table")
	}
	for name, value := range servers {
		m, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid server table")
		}
		if m["url"] != nil {
			continue
		}
		cmd, ok := m["command"].(string)
		if !ok || cmd == "" {
			continue
		}
		l := Launcher{Command: cmd, Args: []string{}}
		if a, exists := m["args"]; exists {
			l.HadArgs = true
			array, ok := a.([]any)
			if !ok {
				return fmt.Errorf("args must be a string array")
			}
			for _, v := range array {
				s, ok := v.(string)
				if !ok {
					return fmt.Errorf("args must be strings")
				}
				l.Args = append(l.Args, s)
			}
		}
		d.entries[name] = l
	}
	var p unstable.Parser
	p.Reset(d.data)
	var table []string
	for p.NextExpression() {
		n := p.Expression()
		if n.Kind == unstable.Table || n.Kind == unstable.ArrayTable {
			table = nil
			it := n.Key()
			for it.Next() {
				table = append(table, string(it.Node().Data))
			}
			continue
		}
		if n.Kind != unstable.KeyValue || len(table) != 2 || table[0] != "mcp_servers" {
			continue
		}
		it := n.Key()
		var keys []*unstable.Node
		for it.Next() {
			keys = append(keys, it.Node())
		}
		if len(keys) != 1 {
			continue
		}
		key := string(keys[0].Data)
		if key != "command" && key != "args" {
			continue
		}
		start := int(keys[0].Raw.Offset + keys[0].Raw.Length)
		for start < len(d.data) && d.data[start] != '=' {
			start++
		}
		start++
		for start < len(d.data) && strings.ContainsRune(" \t", rune(d.data[start])) {
			start++
		}
		end := tomlValueEnd(d.data, start)
		name := table[1]
		if d.fields[name] == nil {
			d.fields[name] = map[string]span{}
		}
		d.fields[name][key] = span{start, end}
		if key == "command" {
			at := end
			for at < len(d.data) && d.data[at] != '\n' {
				at++
			}
			if at < len(d.data) {
				at++
			}
			d.insert[name] = at
		}
	}
	if p.Error() != nil {
		return fmt.Errorf("invalid TOML syntax")
	}
	return nil
}

// The full TOML parser validates syntax first; this scanner only locates the
// end of string/array launcher values, including multiline arrays/comments.
func tomlValueEnd(b []byte, start int) int {
	depth := 0
	var quote byte
	triple, escaped, comment := false, false, false
	for i := start; i < len(b); i++ {
		c := b[i]
		if comment {
			if c == '\n' {
				comment = false
			}
			continue
		}
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if quote == '"' && c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				if triple {
					if i+2 < len(b) && b[i+1] == quote && b[i+2] == quote {
						i += 2
						quote = 0
					}
				} else {
					quote = 0
				}
				if quote == 0 && depth == 0 {
					return i + 1
				}
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			triple = i+2 < len(b) && b[i+1] == c && b[i+2] == c
			if triple {
				i += 2
			}
			continue
		}
		if c == '#' {
			comment = true
			continue
		}
		if c == '[' {
			depth++
		}
		if c == ']' {
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(b)
}
