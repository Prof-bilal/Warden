package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

func printValidateHelp() {
	fmt.Fprintln(os.Stderr, "Usage: warden validate <policy-file> [policy-file...]")
	fmt.Fprintln(os.Stderr, "Check policies with the same loader used by warden run.")
}

func cmdValidate(args []string) error {
	flags := flag.NewFlagSet("warden validate", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Usage = printValidateHelp
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() == 0 {
		return fmt.Errorf("supply at least one policy file")
	}
	var failures []error
	for _, path := range flags.Args() {
		p, _, _, err := resolvePolicy(policySource{path: path})
		if err != nil {
			failures = append(failures, err)
			continue
		}
		for _, advisory := range p.Advisories() {
			fmt.Fprintf(os.Stderr, "warden warning: %q: %s\n", path, advisory)
		}
		fmt.Fprintf(os.Stdout, "Valid policy: %q\n", path)
	}
	return errors.Join(failures...)
}
