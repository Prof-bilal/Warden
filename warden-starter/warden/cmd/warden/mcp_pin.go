package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/mcpbridge"
)

func cmdMCPPin(args []string) error {
	flags := flag.NewFlagSet("warden mcp pin", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	server := flags.String("server", "", "Bridge server name")
	diff := flags.String("diff", "", "Show the accepted and observed hashes for a tool")
	accept := flags.String("accept", "", "Review and accept the observed hash for a tool")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *server == "" || (*diff == "") == (*accept == "") || flags.NArg() != 0 {
		return fmt.Errorf("usage: warden mcp pin --server <name> (--diff <tool> | --accept <tool>)")
	}
	store, err := mcpbridge.DefaultPinStore()
	if err != nil {
		return err
	}
	tool := *diff
	if tool == "" {
		tool = *accept
	}
	pin, err := store.Status(*server, tool)
	if err != nil {
		return err
	}
	if pin.Accepted == "" {
		return fmt.Errorf("tool %q has no pin for server %q", tool, *server)
	}
	message := fmt.Sprintf("Server %q, tool %q\nAccepted SHA-256: %s", *server, tool, pin.Accepted)
	if pin.Observed != "" {
		message += "\nObserved SHA-256: " + pin.Observed
	} else {
		message += "\nNo changed hash recorded."
	}
	if *diff != "" {
		_, err := fmt.Fprintln(os.Stdout, message)
		return err
	}
	if pin.Observed == "" {
		return fmt.Errorf("tool %q has no changed hash to accept", tool)
	}
	ok, err := approve.NewPrompter("", time.Minute, nil).Confirm(message)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("tool pin change declined")
	}
	if err := store.Accept(*server, tool, pin.Accepted, pin.Observed); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Accepted SHA-256 for %q: %s\n", tool, pin.Observed)
	return nil
}
