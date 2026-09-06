//go:build unix

package approve

import "os"

// openConsole opens the controlling terminal for prompting. Using /dev/tty
// (instead of stdin/stdout/stderr) keeps the MCP stdio channel
// protocol-clean: the client never sees approval chatter.
func openConsole() (*os.File, error) {
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}
