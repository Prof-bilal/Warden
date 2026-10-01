// Package clientconfig implements narrow, reversible launcher edits.
package clientconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Client struct {
	ID, Name, Format string
	Scopes           []string
}

var Clients = []Client{
	{"claude-desktop", "Claude Desktop", "json", []string{"user"}},
	{"claude-code", "Claude Code", "json", []string{"project", "user"}},
	{"cursor", "Cursor", "json", []string{"project", "user"}},
	{"codex", "Codex", "toml", []string{"project", "user"}},
	{"vscode", "VS Code", "jsonc", []string{"project"}},
	{"gemini", "Gemini CLI", "json", []string{"project", "user"}},
	{"cline", "Cline CLI / extension", "json", []string{"user", "explicit"}},
	{"cascade", "Legacy Cascade", "json", []string{"explicit"}},
	{"generic", "Other MCP host", "json", []string{"explicit"}},
}

func Get(id string) (Client, error) {
	for _, c := range Clients {
		if c.ID == id {
			return c, nil
		}
	}
	return Client{}, fmt.Errorf("unknown client %q; use warden clients", id)
}

func ConfigPath(id, scope, explicit string) (string, error) {
	c, err := Get(id)
	if err != nil {
		return "", err
	}
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	valid := false
	for _, s := range c.Scopes {
		if s == scope {
			valid = true
		}
	}
	if !valid {
		return "", fmt.Errorf("%s requires a supported scope or --config", id)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	base, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if scope == "user" {
		base = home
	}
	switch id {
	case "cline":
		if scope == "explicit" {
			return "", fmt.Errorf("extension configuration requires --config")
		}
		return filepath.Join(home, ".cline", "mcp.json"), nil
	case "cursor":
		return filepath.Join(base, ".cursor", "mcp.json"), nil
	case "codex":
		return filepath.Join(base, ".codex", "config.toml"), nil
	case "gemini":
		return filepath.Join(base, ".gemini", "settings.json"), nil
	case "vscode":
		return filepath.Join(base, ".vscode", "mcp.json"), nil
	case "claude-code":
		if scope == "user" {
			return filepath.Join(home, ".claude.json"), nil
		}
		return filepath.Join(base, ".mcp.json"), nil
	case "claude-desktop":
		switch runtime.GOOS {
		case "darwin":
			return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
		case "windows":
			appdata := os.Getenv("APPDATA")
			if appdata == "" {
				return "", fmt.Errorf("APPDATA is unset; supply --config")
			}
			return filepath.Join(appdata, "Claude", "claude_desktop_config.json"), nil
		default:
			return "", fmt.Errorf("Claude Desktop has no supported default path on this OS; supply --config")
		}
	default:
		return "", fmt.Errorf("supply --config for this host")
	}
}
