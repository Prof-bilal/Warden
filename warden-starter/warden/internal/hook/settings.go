package hook

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/warden-sandbox/warden/internal/mcpbridge"
)

func AddSettings(raw []byte, executable, policyPath string) ([]byte, error) {
	for _, path := range []string{executable, policyPath} {
		if !filepath.IsAbs(path) || strings.Contains(path, "${") {
			return nil, fmt.Errorf("hook paths must be absolute without interpolation placeholders")
		}
	}
	var settings map[string]json.RawMessage
	if mcpbridge.StrictJSON(raw) != nil || json.Unmarshal(raw, &settings) != nil || settings == nil {
		return nil, fmt.Errorf("invalid Claude Code settings JSON")
	}
	if string(settings["disableAllHooks"]) == "true" {
		return nil, fmt.Errorf("Claude Code hooks are disabled in these settings")
	}
	hooks := make(map[string]json.RawMessage)
	if data, ok := settings["hooks"]; ok && (json.Unmarshal(data, &hooks) != nil || hooks == nil) {
		return nil, fmt.Errorf("Claude Code hooks must be an object")
	}
	var preToolUse []json.RawMessage
	if data, ok := hooks["PreToolUse"]; ok && (json.Unmarshal(data, &preToolUse) != nil || preToolUse == nil) {
		return nil, fmt.Errorf("Claude Code PreToolUse hooks must be an array")
	}
	entry, err := json.Marshal(map[string]any{
		"matcher": Matcher,
		"hooks": []map[string]any{{
			"type": "command", "command": executable, "args": []string{"hook", "check", "--policy", policyPath},
			"onFailure": "block", "timeout": 5,
		}},
	})
	if err != nil {
		return nil, err
	}
	hooks["PreToolUse"], err = json.Marshal(append(preToolUse, entry))
	if err != nil {
		return nil, err
	}
	settings["hooks"], err = json.Marshal(hooks)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
