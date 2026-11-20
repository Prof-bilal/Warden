package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/maintenance"
	"github.com/warden-sandbox/warden/internal/packs"
	"github.com/warden-sandbox/warden/internal/policy"
	"gopkg.in/yaml.v3"
)

var errNoRunPolicy = errors.New("no policy found in the working directory; supply --policy <file>, create warden.yaml, or use warden learn")

func runPolicyPath(explicit string, hasCommand bool) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if !hasCommand {
		return "", fmt.Errorf("pass a command or supply --policy <file> with a command")
	}

	const local = "warden.yaml"
	info, err := os.Lstat(local)
	if os.IsNotExist(err) {
		return "", errNoRunPolicy
	}
	if err != nil {
		return "", fmt.Errorf("inspect %s: %w", local, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s must be a regular file; supply --policy <file> to select another policy", local)
	}
	return local, nil
}

func resolveRunPack(cmd []string, backend string) (path, digest string, err error) {
	if len(cmd) < 2 || len(cmd) > 3 || cmd[0] != "npx" {
		return "", "", errNoRunPolicy
	}
	name, version, ok := splitNPMSpec(cmd[1])
	if !ok {
		return "", "", errNoRunPolicy
	}
	if version == "" {
		filesystem, err := packs.Find("filesystem")
		if err != nil || name != filesystem.Artifact {
			return "", "", errNoRunPolicy
		}
		// The bare launcher names the bundled version, never npm's latest tag.
		version = filesystem.Version
	}
	pack, ok := packs.FindByArtifact(name, version)
	if !ok {
		return "", "", errNoRunPolicy
	}
	dataPath := ""
	if len(cmd) == 3 {
		if pack.ID != "filesystem" || cmd[2] == "" || strings.HasPrefix(cmd[2], "-") {
			return "", "", errNoRunPolicy
		}
		dataPath = cmd[2]
	}
	if pack.PathMode == "write" {
		return "", "", fmt.Errorf("pack %q write access %s; supply an explicit --policy", pack.ID, classifyGrant("write", "", nil))
	}
	for _, name := range pack.Env {
		if classifyGrant("env", name, nil) != "" {
			return "", "", fmt.Errorf("pack %q requires credential %s; supply an explicit --policy", pack.ID, name)
		}
	}
	for _, host := range pack.Hosts {
		if reason := classifyGrant("host", host, pack.Hosts); reason != "" {
			return "", "", fmt.Errorf("pack %q host %q: %s; supply an explicit --policy", pack.ID, host, reason)
		}
	}
	if backend == "" {
		backend = "auto"
	}
	if !validSetupBackend(backend) {
		return "", "", fmt.Errorf("invalid backend")
	}

	runtimeDir := defaultRuntimeDir(pack.ID, pack.Version)
	if err := os.MkdirAll(filepath.Dir(runtimeDir), 0700); err != nil {
		return "", "", err
	}
	src, _, err := resolveFromPack(pack, runtimeDir, dataPath, backend)
	if err != nil {
		return "", "", err
	}
	digest = fmt.Sprintf("%x", sha256.Sum256(src.policyBody))
	path = filepath.Join(runtimeDir, "run-"+digest+".yaml")
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		if err := reviewRunPackUpgrade(runtimeDir, src.policyBody, digest); err != nil {
			return "", "", err
		}
	}
	if err := writeAddPolicy(path, src.policyBody); err != nil {
		return "", "", err
	}
	return path, digest, nil
}

func reviewRunPackUpgrade(runtimeDir string, body []byte, digest string) error {
	versions, err := os.ReadDir(filepath.Dir(runtimeDir))
	if err != nil {
		return err
	}
	var previous string
	var newest time.Time
	for _, version := range versions {
		if !version.IsDir() || version.Name() == filepath.Base(runtimeDir) {
			continue
		}
		dir := filepath.Join(filepath.Dir(runtimeDir), version.Name())
		files, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, file := range files {
			name := file.Name()
			if !strings.HasPrefix(name, "run-") || !strings.HasSuffix(name, ".yaml") {
				continue
			}
			info, err := file.Info()
			if err != nil {
				return err
			}
			if info.Mode().IsRegular() && (previous == "" || info.ModTime().After(newest)) {
				previous, newest = filepath.Join(dir, name), info.ModTime()
			}
		}
	}
	if previous == "" {
		return nil
	}
	pin := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(previous), "run-"), ".yaml")
	before, _, _, err := resolvePolicy(policySource{path: previous, pin: pin, provenance: "pack"})
	if err != nil {
		return fmt.Errorf("read previous pack policy: %w", err)
	}
	var after policy.Policy
	if err := yaml.Unmarshal(body, &after); err != nil {
		return err
	}
	after.Normalize()
	var question strings.Builder
	fmt.Fprintf(&question, "Review the pack version change from %q to %q.\nPolicy SHA-256: %s\n",
		filepath.Base(filepath.Dir(previous)), filepath.Base(runtimeDir), digest)
	if err := writePolicyDiffMarkdown(&question, maintenance.Compare(before, after)); err != nil {
		return err
	}
	if stdin, err := os.Stdin.Stat(); err != nil || stdin.Mode()&os.ModeCharDevice == 0 || approve.CheckTTY() != nil {
		return fmt.Errorf("%sPack upgrade requires an interactive review or an explicit --policy", question.String())
	}
	ok, err := approve.NewPrompter("", time.Minute, nil).Confirm(question.String())
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("pack upgrade declined")
	}
	return nil
}

func resolveRunWizard(cmd []string, backend string) (string, string, error) {
	stdin, err := os.Stdin.Stat()
	if err != nil || stdin.Mode()&os.ModeCharDevice == 0 || approve.CheckTTY() != nil {
		return "", "", errNoRunPolicy
	}
	if backend != "" && !validSetupBackend(backend) {
		return "", "", fmt.Errorf("invalid backend")
	}
	cmd, err = policy.ResolveExecutable(cmd)
	if err != nil {
		return "", "", err
	}
	cmd[0], err = filepath.Abs(cmd[0])
	if err != nil {
		return "", "", err
	}
	if err := checkSandboxCommand(cmd); err != nil {
		return "", "", err
	}

	body, digest := wizardPolicy(cmd)
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		config = filepath.Join(home, ".config")
	}
	if !filepath.IsAbs(config) {
		return "", "", fmt.Errorf("policy configuration directory must be absolute")
	}
	dir := filepath.Join(config, "warden", "policies")
	path := filepath.Join(dir, digest+".yaml")
	prompt := fmt.Sprintf("Review starter policy for %q\nNo additional filesystem, network or environment grants.\nSave to %q\nPolicy SHA-256: %s", cmd, path, digest)
	ok, err := approve.NewPrompter("", time.Minute, nil).Confirm(prompt)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", fmt.Errorf("starter policy declined; supply --policy <file> to select an existing policy")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", "", err
	}
	if err := writeAddPolicy(path, body); err != nil {
		return "", "", err
	}
	return path, digest, nil
}

func wizardPolicy(cmd []string) ([]byte, string) {
	argv, _ := json.Marshal(cmd)
	body := []byte("command: " + string(argv) + "\nfilesystem: {protect_secrets: true}\n")
	return body, fmt.Sprintf("%x", sha256.Sum256(body))
}

type policySource struct {
	path       string
	pin        string
	provenance string
	generated  []byte
}

// resolvePolicy validates and hashes the same bytes, with provenance supplied by the caller.
func resolvePolicy(source policySource) (p policy.Policy, digest, provenance string, err error) {
	if source.generated == nil {
		p, digest, err = policy.LoadVerifiedWithDigest(source.path, source.pin)
	} else {
		digest = fmt.Sprintf("%x", sha256.Sum256(source.generated))
		if source.pin != "" && source.pin != digest {
			return policy.Policy{}, "", "", fmt.Errorf("generated policy does not match the expected digest")
		}
		p, err = policy.Parse(source.path, source.generated)
	}
	if err != nil {
		return policy.Policy{}, "", "", err
	}
	provenance = source.provenance
	if provenance == "" {
		provenance = "file"
	}
	return p, digest, provenance, nil
}

func parsePolicyDigest(args []string) (string, []string, error) {
	var digest string
	out := []string{}
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			out = append(out, args[i:]...)
			break
		}
		value := ""
		if args[i] == "--policy-sha256" {
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--policy-sha256 requires a digest")
			}
			i++
			value = args[i]
		} else if strings.HasPrefix(args[i], "--policy-sha256=") {
			value = strings.TrimPrefix(args[i], "--policy-sha256=")
		} else {
			out = append(out, args[i])
			continue
		}
		raw, e := hex.DecodeString(value)
		if e != nil || len(raw) != 32 || digest != "" {
			return "", nil, fmt.Errorf("policy digest must be one SHA-256 value")
		}
		digest = strings.ToLower(value)
	}
	return digest, out, nil
}
