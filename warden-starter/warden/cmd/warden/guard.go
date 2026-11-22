package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/guard"
)

func printGuardHelp() {
	fmt.Fprintln(os.Stderr, "Usage: warden guard npm install <package[@version]> [--output <new-directory>] [--backend auto] [--policy <file>] [--strict] [--override-age]")
	fmt.Fprintln(os.Stderr, "       warden guard pip install <package[==version]> [--output <new-directory>] [--backend auto] [--policy <file>] [--strict] [--override-age]")
	fmt.Fprintln(os.Stderr, "Review and install a release into a dedicated sandboxed runtime. Pip installs reviewed wheels individually.")
}

func cmdGuard(args []string) error {
	if len(args) < 3 || (args[0] != "npm" && args[0] != "pip") || args[1] != "install" {
		return fmt.Errorf("expected npm or pip install <package>")
	}
	flags := flag.NewFlagSet("warden guard "+args[0]+" install", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	output := flags.String("output", "", "new runtime directory")
	backend := flags.String("backend", "auto", "installation sandbox")
	policyPath := flags.String("policy", "", "policy file with additional install guard rules")
	strict := flags.Bool("strict", false, "block missing release metadata")
	override := flags.Bool("override-age", false, "override the cooldown for this invocation and audit the request")
	if err := flags.Parse(args[3:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || !validSetupBackend(*backend) {
		return fmt.Errorf("unexpected arguments or invalid backend")
	}
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		config = filepath.Join(home, ".config")
	}
	if !filepath.IsAbs(config) {
		return fmt.Errorf("guard configuration directory must be absolute")
	}
	rules, err := guard.ReadReleaseRules(filepath.Join(config, "warden", "guard.yaml"))
	if err != nil {
		return err
	}
	rules.StrictMetadata = rules.StrictMetadata || *strict
	rules.OverrideAge = *override
	if *policyPath != "" {
		if err := applyInstallGuardPolicy(&rules, *policyPath); err != nil {
			return err
		}
	}
	if args[0] == "pip" {
		name, version, pinned := strings.Cut(args[2], "==")
		if pinned && version == "" {
			return fmt.Errorf("empty Python package version")
		}
		return guardPipInstall(context.Background(), "https://pypi.org", name, version, *output, *backend, time.Now(), rules)
	}
	name, version, ok := splitNPMSpec(args[2])
	if !ok || strings.HasSuffix(args[2], "@") {
		return fmt.Errorf("invalid npm package specification %q", args[2])
	}
	return guardNPMInstall(context.Background(), "https://registry.npmjs.org", name, version, *output, *backend, time.Now(), rules, installNPMRuntime)
}

func applyInstallGuardPolicy(rules *guard.ReleaseRules, path string) error {
	p, _, _, err := resolvePolicy(policySource{path: path})
	if err != nil {
		return err
	}
	if p.Guard != nil {
		rules.MinimumAge = max(rules.MinimumAge, p.Guard.MinimumReleaseAge)
		rules.StrictMetadata = rules.StrictMetadata || p.Guard.Strict
	}
	return nil
}

func guardNPMInstall(ctx context.Context, registry, name, version, output, backend string, now time.Time, rules guard.ReleaseRules,
	install func(string, string, string, []byte, string) error) error {
	release, archive, decision, err := guard.ReviewNPMInstall(ctx, nil, registry, name, version, now, rules)
	if err != nil {
		return err
	}
	spec := release.Facts.Name + "@" + release.Facts.Version
	if err := reportGuardDecision("npm", spec, decision, rules.OverrideAge); err != nil {
		return err
	}
	if len(archive) == 0 {
		return fmt.Errorf("reviewed npm archive is missing")
	}
	output, err = guardOutput("npm", spec, output)
	if err != nil {
		return err
	}
	var scriptManifest string
	if slices.Contains(rules.ScriptAllowlist, spec) && (len(release.Facts.InstallScripts) != 0 || release.Facts.HasBindingGYP) {
		scriptManifest = release.Facts.NPMManifestSHA256
		if scriptManifest == "" {
			return fmt.Errorf("reviewed lifecycle manifest is missing")
		}
	}
	if err := install(spec, output, backend, archive, scriptManifest); err != nil {
		return err
	}
	if scriptManifest != "" {
		fmt.Fprintf(os.Stdout, "Installed %s in %s with lifecycle approval scoped to that package.\n", spec, output)
	} else {
		fmt.Fprintf(os.Stdout, "Installed %s in %s with lifecycle scripts disabled.\n", spec, output)
	}
	return nil
}

func reportGuardDecision(kind, spec string, decision guard.ReleaseDecision, override bool) error {
	for _, warning := range decision.Warnings {
		fmt.Fprintln(os.Stderr, "warden guard warning:", warning)
	}
	if override {
		log, _, err := audit.OpenDefault()
		if err != nil {
			return err
		}
		err = audit.New(log).Log(audit.Event{Type: "guard", Kind: kind, Action: "override-age", Resource: spec,
			Allowed: len(decision.Blocks) == 0, Reason: "explicit invocation override requested"})
		closeErr := log.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if len(decision.Blocks) > 0 {
		return fmt.Errorf("blocked %q: %s", spec, strings.Join(decision.Blocks, "; "))
	}
	return nil
}

func guardOutput(kind, spec, output string) (string, error) {
	if output == "" {
		output = filepath.Join(wardenDataHome(), "warden", "guard", kind, filepath.FromSlash(spec))
		if err := os.MkdirAll(filepath.Dir(output), 0700); err != nil {
			return "", err
		}
	}
	return filepath.Abs(output)
}
