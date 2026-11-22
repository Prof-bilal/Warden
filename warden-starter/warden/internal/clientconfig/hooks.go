package clientconfig

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

const hookEntry = "warden-pre-tool-use"
const hookFormat = "hook-settings"

func HooksPath(id, scope, explicit string) (string, error) {
	client, err := Get(id)
	if err != nil {
		return "", err
	}
	if client.HookSettings == "" {
		return "", fmt.Errorf("client %q does not support managed hooks", id)
	}
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	switch scope {
	case "project":
		return filepath.Abs(filepath.Join(filepath.Dir(client.HookSettings), "settings.local.json"))
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, client.HookSettings), nil
	default:
		return "", fmt.Errorf("hook scope must be project or user")
	}
}

func PrepareHook(config, policyPath, digest, executable string, edit func([]byte) ([]byte, error)) (Plan, error) {
	if err := os.MkdirAll(filepath.Dir(config), 0700); err != nil {
		return Plan{}, err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(config))
	if err != nil {
		return Plan{}, err
	}
	config = filepath.Join(parent, filepath.Base(config))
	if stateDir(config) == "" || config == policyPath {
		return Plan{}, fmt.Errorf("hook settings require private undo storage and a separate policy file")
	}
	before, err := ReadRegular(config)
	if err != nil && !os.IsNotExist(err) {
		return Plan{}, err
	}
	p := Plan{Config: config, Server: hookEntry, Format: hookFormat, Policy: policyPath, PolicyHash: digest,
		Before: before, EntryCreated: true, ConfigMissing: os.IsNotExist(err), Wrapped: Launcher{Command: executable}}
	r, err := loadRecord(config, hookEntry)
	if err == nil && r.Active {
		if r.Policy == policyPath && r.Wrapped.Command == executable && Hash(before) == r.AfterHash {
			p.Unchanged = true
			return p, nil
		}
		return Plan{}, fmt.Errorf("managed hook settings differ; uninstall before replacing the hook")
	}
	if err != nil && !os.IsNotExist(err) {
		return Plan{}, err
	}
	input := before
	if p.ConfigMissing {
		input = []byte("{}")
	}
	p.After, err = edit(input)
	return p, err
}

func UndoHook(config string) (bool, error) {
	return Undo(config, hookEntry, false)
}

func undoHookSettings(config string, r Record, dryRun bool) (bool, error) {
	if r.ConfigMissing {
		return undoCreated(config, r, dryRun)
	}
	current, err := ReadRegular(config)
	if err != nil {
		return false, err
	}
	if Hash(current) == r.BeforeHash {
		if !dryRun {
			err = markRecordInactive(config, r)
		}
		return false, err
	}
	if Hash(current) != r.AfterHash {
		return false, fmt.Errorf("settings changed since hook install; refusing to overwrite them")
	}
	backup, err := ReadRegular(r.Backup)
	if err != nil || Hash(backup) != r.BeforeHash {
		return false, fmt.Errorf("hook settings backup missing or modified")
	}
	if dryRun {
		return true, nil
	}
	latest, err := ReadRegular(config)
	if err != nil || !bytes.Equal(latest, current) {
		return false, fmt.Errorf("settings changed during uninstall")
	}
	if err := AtomicWrite(config, backup); err != nil {
		return false, err
	}
	return true, markRecordInactive(config, r)
}
