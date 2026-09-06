//go:build windows

package approve

import "os"

// openConsole opens the Windows console for prompting, keeping the MCP
// stdio handles protocol-clean.
func openConsole() (*os.File, error) {
	return os.OpenFile("CON", os.O_RDWR, 0)
}
