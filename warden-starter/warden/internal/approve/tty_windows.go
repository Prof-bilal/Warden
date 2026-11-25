//go:build windows

package approve

import "os"

// openConsole opens the Windows console for prompting, keeping the MCP
// stdio handles protocol-clean.
func openConsole() (console, error) {
	input, err := os.Open("CONIN$")
	if err != nil {
		return console{}, err
	}
	output, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if err != nil {
		input.Close()
		return console{}, err
	}
	return console{input: input, output: output}, nil
}
