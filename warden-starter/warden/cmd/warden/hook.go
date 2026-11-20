package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/warden-sandbox/warden/internal/clientconfig"
	"github.com/warden-sandbox/warden/internal/hook"
)

func printHookHelp() {
	fmt.Fprintln(os.Stderr, "Usage: warden hook check --policy <file>")
	fmt.Fprintln(os.Stderr, "       warden hook install --policy <file> [--scope project|user] [--config <settings.json>]")
	fmt.Fprintln(os.Stderr, "       warden hook uninstall [--scope project|user] [--config <settings.json>]")
	fmt.Fprintln(os.Stderr, "Manage advisory Claude Code PreToolUse checks.")
}

func cmdHook(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("expected hook check, install, or uninstall")
	}
	flags := flag.NewFlagSet("warden hook "+args[0], flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	explicit := flags.String("policy", "", "Warden policy file")
	scope := flags.String("scope", "project", "Claude Code settings scope")
	config := flags.String("config", "", "explicit Claude Code settings path")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected hook arguments")
	}
	switch args[0] {
	case "check":
		if *explicit == "" {
			return fmt.Errorf("--policy required")
		}
		p, _, _, err := resolvePolicy(policySource{path: *explicit})
		if err != nil {
			return fmt.Errorf("cannot load Warden hook policy")
		}
		return hook.Check(os.Stdin, p, canonicalOutput)
	case "install":
		if *explicit == "" {
			return fmt.Errorf("--policy required")
		}
		policyPath, err := filepath.Abs(*explicit)
		if err != nil {
			return err
		}
		_, digest, _, err := resolvePolicy(policySource{path: policyPath})
		if err != nil {
			return err
		}
		settings, err := clientconfig.HooksPath("claude-code", *scope, *config)
		if err != nil {
			return err
		}
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		plan, err := clientconfig.PrepareHook(settings, policyPath, digest, exe, func(raw []byte) ([]byte, error) {
			return hook.AddSettings(raw, exe, policyPath)
		})
		if err != nil {
			return err
		}
		if _, err := clientconfig.Apply(plan); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Warden PreToolUse hook configured in %q.\n", plan.Config)
		return nil
	case "uninstall":
		settings, err := clientconfig.HooksPath("claude-code", *scope, *config)
		if err != nil {
			return err
		}
		changed, err := clientconfig.UndoHook(settings)
		if err != nil {
			return err
		}
		if changed {
			fmt.Fprintln(os.Stdout, "Warden PreToolUse hook removed.")
		}
		return nil
	default:
		return fmt.Errorf("expected hook check, install, or uninstall")
	}
}
