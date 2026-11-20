package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/warden-sandbox/warden/internal/clientconfig"
	"github.com/warden-sandbox/warden/internal/policy"
	"gopkg.in/yaml.v3"
)

func printMigrateHelp() {
	fmt.Fprintln(os.Stderr, "Usage: warden migrate [--write] [--client ID --config FILE] [policy.yaml ...]")
	fmt.Fprintln(os.Stderr, "Preview policy upgrades and MCP launcher findings. With no policy paths, inspect ./warden.yaml if present.")
	fmt.Fprintln(os.Stderr, "Exit 3 means work remains. --write saves a .bak before an eligible change.")
}

type migrationConfig struct{ path, format string }

// Keep diagnostics about unknown keys without printing user-supplied values.
var unknownMigrationKey = regexp.MustCompile(`(?m)line [0-9]+: field [A-Za-z_][A-Za-z0-9_]* not found`)

func cmdMigrate(args []string, out io.Writer) (int, error) {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	write := fs.Bool("write", false, "apply policy upgrades")
	client := fs.String("client", "", "client ID for an explicit config")
	config := fs.String("config", "", "explicit client config file")
	if err := fs.Parse(args); err != nil {
		return 2, err
	}
	configs, err := migrationConfigs(*client, *config)
	if err != nil {
		return 2, err
	}
	pins, uncertain, pending := scanMigrationConfigs(configs, *config != "", out)
	var edits []clientconfig.MigrationEdit
	var repins []migrationRepin
	for _, pin := range pins {
		if pin.policy == "" || !filepath.IsAbs(pin.policy) {
			fmt.Fprintf(out, "%s: review digest pin with a relative or implicit policy path\n", pin.location)
			pending = true
		}
	}
	paths := fs.Args()
	if len(paths) == 0 {
		if _, err := os.Lstat("warden.yaml"); !os.IsNotExist(err) {
			paths = []string{"warden.yaml"}
		}
	}
	seen := map[string]bool{}
	for _, name := range paths {
		path, err := filepath.Abs(name)
		if err != nil {
			return 2, err
		}
		identity := canonicalMigrationPath(path)
		if seen[identity] {
			continue
		}
		seen[identity] = true
		before, err := clientconfig.ReadRegular(path)
		if err != nil {
			fmt.Fprintf(out, "%q: review required: could not read policy: %v\n", path, err)
			pending = true
			continue
		}
		p, after, err := migratePolicy(path, before)
		if err != nil {
			fmt.Fprintf(out, "%q: review required: %v\n", path, err)
			pending = true
			continue
		}
		if p.MCP != nil && len(p.MCP.AllowTools) == 0 {
			fmt.Fprintf(out, "%q: empty mcp.allow_tools denies every tool; review the list explicitly\n", path)
			pending = true
		}
		blocked := uncertain
		var policyRepins []migrationRepin
		oldHash := clientconfig.Hash(before)
		newHash := oldHash
		if after != nil {
			newHash = clientconfig.Hash(after)
			fmt.Fprintf(out, "%q\n- version: %d\n+ version: 2\n- sha256: %s\n+ sha256: %s\n", path, p.Version, oldHash, newHash)
		}
		for _, pin := range pins {
			if !filepath.IsAbs(pin.policy) {
				blocked = true
				continue
			}
			if canonicalMigrationPath(pin.policy) != identity {
				continue
			}
			if after == nil && pin.digest == oldHash {
				continue
			}
			if pin.digest != oldHash {
				if after != nil || !recoverableMigrationPin(path, before, pin.digest) {
					fmt.Fprintf(out, "%s: launcher digest does not match the existing policy; review without replacing its pin\n", pin.location)
					blocked = true
					pending = true
					continue
				}
				fmt.Fprintf(out, "%s: resume a pin update verified against the policy backup\n", pin.location)
			}
			policyRepins = append(policyRepins, migrationRepin{pin, newHash})
		}
		if blocked && (after != nil || len(policyRepins) > 0) {
			pending = true
			continue
		}
		if after != nil {
			edits = append(edits, clientconfig.MigrationEdit{Path: identity, Before: before, After: after})
		} else if len(policyRepins) > 0 {
			edits = append(edits, clientconfig.MigrationEdit{Path: identity, Before: before, After: before})
		}
		repins = append(repins, policyRepins...)
	}
	pinEdits, err := planMigrationRepins(repins, out)
	if err != nil {
		fmt.Fprintf(out, "Review required: %v\n", err)
		return 3, nil
	}
	edits = append(edits, pinEdits...)
	if len(edits) > 0 {
		if !*write {
			return 3, nil
		}
		if err := clientconfig.ApplyMigration(edits); err != nil {
			return 2, err
		}
		for _, edit := range edits {
			if !bytes.Equal(edit.Before, edit.After) {
				fmt.Fprintf(out, "Migrated %q (backup %q)\n", edit.Path, edit.Path+".bak")
			}
		}
	}
	if pending {
		return 3, nil
	}
	return 0, nil
}

func recoverableMigrationPin(path string, current []byte, oldPin string) bool {
	backup, err := clientconfig.ReadRegular(path + ".bak")
	if err != nil || clientconfig.Hash(backup) != oldPin {
		return false
	}
	_, migrated, err := migratePolicy(path, backup)
	return err == nil && bytes.Equal(current, migrated)
}

func migratePolicy(path string, raw []byte) (policy.Policy, []byte, error) {
	var doc, extra yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&doc); err != nil || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return policy.Policy{}, nil, fmt.Errorf("expected one YAML policy mapping")
	}
	if err := decoder.Decode(&extra); err != io.EOF {
		return policy.Policy{}, nil, fmt.Errorf("expected one YAML document")
	}
	p, err := policy.Parse(path, raw)
	if err != nil {
		if unknown := unknownMigrationKey.FindAllString(err.Error(), -1); len(unknown) > 0 {
			return p, nil, fmt.Errorf("unsupported policy keys: %s", strings.Join(unknown, "; "))
		}
		return p, nil, fmt.Errorf("invalid or unsupported policy; inspect it with warden validate")
	}
	if p.Version == 2 {
		return p, nil, nil
	}
	mapping := doc.Content[0]
	version := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "2"}
	found := false
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == "version" {
			old := mapping.Content[i+1]
			// An alias or anchored value may also be used elsewhere in the policy.
			if old.Kind != yaml.ScalarNode || old.Anchor != "" {
				return p, nil, fmt.Errorf("version uses an alias or anchor")
			}
			old.Tag, old.Value, old.Style = version.Tag, version.Value, 0
			found = true
			break
		}
	}
	if !found {
		mapping.Content = append([]*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "version"}, version}, mapping.Content...)
	}
	after, err := yaml.Marshal(&doc)
	if err != nil {
		return p, nil, fmt.Errorf("could not encode migrated policy")
	}
	upgraded, err := policy.Parse(path, after)
	if err != nil || upgraded.Version != 2 {
		return p, nil, fmt.Errorf("migrated policy did not validate")
	}
	upgraded.Version = p.Version
	if !reflect.DeepEqual(p, upgraded) {
		return p, nil, fmt.Errorf("migration would change effective policy")
	}
	return p, after, nil
}

func migrationConfigs(client, explicit string) ([]migrationConfig, error) {
	if client != "" || explicit != "" {
		if client == "" || explicit == "" {
			return nil, fmt.Errorf("--client and --config must be supplied together")
		}
		c, err := clientconfig.Get(client)
		if err != nil {
			return nil, err
		}
		path, err := clientconfig.ConfigPath(client, "", explicit)
		if err != nil {
			return nil, err
		}
		return []migrationConfig{{path, c.Format}}, nil
	}
	var configs []migrationConfig
	seen := map[string]bool{}
	for _, c := range clientconfig.Clients {
		for _, scope := range c.Scopes {
			if scope == "explicit" {
				continue
			}
			path, err := clientconfig.ConfigPath(c.ID, scope, "")
			if err == nil && !seen[path] {
				configs = append(configs, migrationConfig{path, c.Format})
				seen[path] = true
			}
		}
	}
	return configs, nil
}

type migrationPin struct {
	policy, digest, location, name string
	config                         migrationConfig
	snapshot                       []byte
	launcher                       clientconfig.Launcher
}

type migrationRepin struct {
	pin  migrationPin
	next string
}

func planMigrationRepins(repins []migrationRepin, out io.Writer) ([]clientconfig.MigrationEdit, error) {
	var edits []clientconfig.MigrationEdit
	byConfig := map[string][]migrationRepin{}
	var paths []string
	for _, repin := range repins {
		path := canonicalMigrationPath(repin.pin.config.path)
		if _, exists := byConfig[path]; !exists {
			paths = append(paths, path)
		}
		byConfig[path] = append(byConfig[path], repin)
	}
	for _, path := range paths {
		group := byConfig[path]
		before := group[0].pin.snapshot
		after := before
		updated := make([]clientconfig.Launcher, 0, len(group))
		for _, repin := range group {
			pin := repin.pin
			doc, err := clientconfig.Parse(after, pin.config.format)
			if err != nil {
				return nil, fmt.Errorf("%s: cannot patch launcher", pin.location)
			}
			launcher := pin.launcher
			launcher.Args = append([]string(nil), launcher.Args...)
			flags, _ := splitRunCommand(launcher.Args[1:])
			for i, arg := range flags {
				if arg == "--policy-sha256" {
					flags[i+1] = repin.next
					break
				}
				if strings.HasPrefix(arg, "--policy-sha256=") {
					flags[i] = "--policy-sha256=" + repin.next
					break
				}
			}
			after, err = doc.Patch(pin.name, launcher)
			if err != nil {
				return nil, fmt.Errorf("%s: cannot patch launcher", pin.location)
			}
			updated = append(updated, launcher)
		}
		for i, repin := range group {
			pin := repin.pin
			edit, err := clientconfig.MigrationRecord(path, pin.name, pin.launcher, updated[i], pin.digest, repin.next, before, after, len(group) == 1)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", pin.location, err)
			}
			if edit != nil {
				edits = append(edits, *edit)
			}
			fmt.Fprintf(out, "%s\n- policy pin: %s\n+ policy pin: %s\n", pin.location, pin.digest, repin.next)
		}
		edits = append(edits, clientconfig.MigrationEdit{Path: path, Before: before, After: after})
	}
	return edits, nil
}

func scanMigrationConfigs(configs []migrationConfig, explicit bool, out io.Writer) ([]migrationPin, bool, bool) {
	var pins []migrationPin
	uncertain, pending := false, false
	for _, config := range configs {
		raw, err := clientconfig.ReadRegular(config.path)
		if !explicit && os.IsNotExist(err) {
			continue
		}
		if err != nil {
			fmt.Fprintf(out, "%q: review required: could not read client config\n", config.path)
			uncertain = true
			continue
		}
		doc, err := clientconfig.Parse(raw, config.format)
		if errors.Is(err, clientconfig.ErrNoServers) {
			continue
		}
		if err != nil {
			fmt.Fprintf(out, "%q: review required: could not parse client config\n", config.path)
			uncertain = true
			continue
		}
		for _, name := range doc.Names() {
			launcher, _ := doc.Launcher(name)
			base := filepath.Base(launcher.Command)
			if !strings.EqualFold(base, "warden") && !strings.EqualFold(base, "warden.exe") || len(launcher.Args) == 0 {
				continue
			}
			location := fmt.Sprintf("%q server %q", config.path, name)
			switch launcher.Args[0] {
			case "run":
				flags, _ := splitRunCommand(launcher.Args[1:])
				pin, flags, err := parsePolicyDigest(flags)
				if err != nil {
					fmt.Fprintf(out, "%s: review invalid policy digest flag\n", location)
					uncertain = true
					continue
				}
				if pin != "" {
					_, _, flags, err := parseApproveFlags(flags)
					if err != nil {
						flags = nil
					}
					path, _, _, _, _ := parseRunArgs(flags)
					pins = append(pins, migrationPin{path, pin, location, name, config, raw, launcher})
				}
			case "proxy":
				if !inspectMigrationProxy(launcher.Args[1:], location, out) {
					pending = true
				}
			}
		}
	}
	return pins, uncertain, pending || uncertain
}

func inspectMigrationProxy(args []string, location string, out io.Writer) bool {
	path, _, upstream, _, unsafe, err := parseProxyArgs(args)
	if err != nil || !filepath.IsAbs(path) {
		fmt.Fprintf(out, "%s: review proxy arguments and policy path\n", location)
		return false
	}
	raw, err := clientconfig.ReadRegular(path)
	if err != nil {
		fmt.Fprintf(out, "%s: review unreadable proxy policy\n", location)
		return false
	}
	p, err := policy.Parse(path, raw)
	if err != nil || p.MCP == nil {
		fmt.Fprintf(out, "%s: review invalid proxy policy\n", location)
		return false
	}
	if upstream == "" {
		upstream = p.MCP.Upstream
	}
	ok := true
	if strings.HasPrefix(upstream, "stdio:") && !unsafe {
		fmt.Fprintf(out, "%s: stdio proxy requires explicit --unsafe-unconfined or a sandboxed launcher\n", location)
		ok = false
	}
	if len(p.MCP.AllowTools) == 0 {
		fmt.Fprintf(out, "%s: empty mcp.allow_tools denies every tool\n", location)
		ok = false
	}
	return ok
}

func canonicalMigrationPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return filepath.Clean(path)
}
