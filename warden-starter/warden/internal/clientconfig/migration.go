package clientconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/warden-sandbox/warden/internal/privatefile"
)

type MigrationEdit struct {
	Path          string
	Before, After []byte
}

// MigrationRecord preserves the undo record when the only launcher change is its policy pin.
func MigrationRecord(config, server string, old, updated Launcher, policyHash, nextHash string, before, after []byte, single bool) (*MigrationEdit, error) {
	path := statePath(config, server)
	r, raw, err := loadRecordSnapshot(config, server)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !r.Active {
		return nil, nil
	}
	if Equal(r.Wrapped, updated) && r.PolicyHash == nextHash {
		return nil, nil
	}
	if !Equal(r.Wrapped, old) || r.PolicyHash != policyHash {
		return nil, fmt.Errorf("launcher or pin has drifted from the undo record")
	}
	if r.ConfigMissing && r.AfterHash != Hash(before) {
		return nil, fmt.Errorf("managed client file has changed since creation")
	}
	r.Wrapped, r.PolicyHash = updated, nextHash
	if r.ConfigMissing || (single && r.AfterHash == Hash(before)) {
		r.AfterHash = Hash(after)
	}
	edited, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}
	return &MigrationEdit{path, raw, edited}, nil
}

// ApplyMigration locks all affected files, creates backups, then swaps the checked snapshots.
func ApplyMigration(edits []MigrationEdit) error {
	paths := make([]string, 0, len(edits))
	seen := map[string]bool{}
	for _, edit := range edits {
		if seen[edit.Path] {
			return fmt.Errorf("duplicate migration target %q", edit.Path)
		}
		seen[edit.Path] = true
		paths = append(paths, edit.Path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if seen[path+".bak"] {
			return fmt.Errorf("backup also selected for migration: %q", path+".bak")
		}
		release, err := lock(path)
		if err != nil {
			return fmt.Errorf("%q: %w", path, err)
		}
		defer release()
	}
	verify := func(edit MigrationEdit, expected []byte) error {
		current, err := ReadRegular(edit.Path)
		if err != nil {
			return err
		}
		if !bytes.Equal(current, expected) {
			return fmt.Errorf("file changed during migration")
		}
		return nil
	}
	for _, edit := range edits {
		if err := verify(edit, edit.Before); err != nil {
			return fmt.Errorf("%q: %w", edit.Path, err)
		}
	}
	for _, edit := range edits {
		if bytes.Equal(edit.Before, edit.After) {
			continue
		}
		backup := edit.Path + ".bak"
		raw, err := ReadRegular(backup)
		switch {
		case os.IsNotExist(err):
			err = writeNew(backup, edit.Before)
		case err == nil && bytes.Equal(raw, edit.Before):
			err = privatefile.Protect(backup, false)
		case err == nil:
			err = fmt.Errorf("existing backup contains different data")
		}
		if err != nil {
			return fmt.Errorf("create backup %q: %w", backup, err)
		}
	}
	for i, edit := range edits {
		err := verify(edit, edit.Before)
		if err == nil && !bytes.Equal(edit.Before, edit.After) {
			err = AtomicWrite(edit.Path, edit.After)
		}
		if err != nil {
			result := fmt.Errorf("%q: %w", edit.Path, err)
			for j := i - 1; j >= 0; j-- {
				applied := edits[j]
				if bytes.Equal(applied.Before, applied.After) {
					continue
				}
				rollbackErr := verify(applied, applied.After)
				if rollbackErr == nil {
					rollbackErr = AtomicWrite(applied.Path, applied.Before)
				}
				if rollbackErr != nil {
					result = errors.Join(result, fmt.Errorf("restore %q from its backup: %w", applied.Path, rollbackErr))
				}
			}
			return result
		}
	}
	return nil
}
