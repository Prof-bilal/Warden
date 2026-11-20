// Package hook implements advisory Claude Code PreToolUse checks.
package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/warden-sandbox/warden/internal/mcpbridge"
	"github.com/warden-sandbox/warden/internal/policy"
)

const Matcher = "Bash|PowerShell|Read|Write|Edit|MultiEdit|NotebookEdit"
const payloadLimit = 4 << 20

var simpleShellWord = regexp.MustCompile(`^[a-zA-Z0-9_./:=+@%,\-]+$`)

// Check rejects malformed input and tool calls outside the advisory policy.
func Check(r io.Reader, p policy.Policy, canonical func(string) (string, error)) error {
	data, err := io.ReadAll(io.LimitReader(r, payloadLimit+1))
	if err != nil || len(data) > payloadLimit || mcpbridge.StrictJSON(data) != nil {
		return fmt.Errorf("invalid PreToolUse payload")
	}
	var event map[string]json.RawMessage
	if json.Unmarshal(data, &event) != nil || field(event, "hook_event_name") != "PreToolUse" {
		return fmt.Errorf("expected a PreToolUse event")
	}
	var input map[string]json.RawMessage
	if json.Unmarshal(event["tool_input"], &input) != nil || input == nil {
		return fmt.Errorf("invalid PreToolUse tool input")
	}
	switch tool := field(event, "tool_name"); tool {
	case "Bash", "PowerShell":
		command := field(input, "command")
		line := strings.Trim(command, " \t")
		argv := strings.FieldsFunc(line, func(r rune) bool { return r == ' ' || r == '\t' })
		if len(argv) == 0 {
			return fmt.Errorf("expected one reviewed shell command")
		}
		for _, arg := range argv {
			if !simpleShellWord.MatchString(arg) {
				return fmt.Errorf("only literal arguments in a single reviewed command are supported")
			}
		}
		reviewed, err := p.ResolveCommand(nil)
		if err != nil || strings.Contains(argv[0], "=") || !slices.Equal(argv, reviewed) {
			return fmt.Errorf("shell command does not match the reviewed policy command")
		}
		return nil
	case "Read", "Write", "Edit", "MultiEdit", "NotebookEdit":
		key := "file_path"
		if tool == "NotebookEdit" {
			key = "notebook_path"
		}
		path := field(input, key)
		if !filepath.IsAbs(path) || strings.ContainsRune(path, 0) {
			return fmt.Errorf("expected an absolute file path")
		}
		resolved, err := canonical(path)
		if err != nil {
			return fmt.Errorf("cannot resolve tool path for review")
		}
		p = p.WithActiveGrants(time.Now())
		if p.DeniesHookPath(filepath.Clean(path)) {
			return fmt.Errorf("file access is covered by a deny rule")
		}
		p.Filesystem.Read, err = canonicalPaths(p.Filesystem.Read, canonical)
		if err != nil {
			return err
		}
		p.Filesystem.Write, err = canonicalPaths(p.Filesystem.Write, canonical)
		if err != nil {
			return err
		}
		p.Filesystem.Deny, err = canonicalPaths(p.Filesystem.Deny, canonical)
		if err != nil {
			return err
		}
		if !p.AllowsHookPath(resolved, tool != "Read") {
			return fmt.Errorf("file access is outside the policy or covered by a deny rule")
		}
		return nil
	default:
		return fmt.Errorf("unsupported PreToolUse tool")
	}
}

func canonicalPaths(paths []string, canonical func(string) (string, error)) ([]string, error) {
	out := make([]string, len(paths))
	for i, path := range paths {
		resolved, err := canonical(path)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve a hook policy path")
		}
		out[i] = resolved
	}
	return out, nil
}

func field(object map[string]json.RawMessage, name string) string {
	var value string
	_ = json.Unmarshal(object[name], &value)
	return value
}
