//go:build !windows

package proxy

import (
	"context"
	"net"
	"syscall"
)

// listenBridge allows the fixed loopback bridge to be rebound immediately
// after a sandboxed child exits. This matters on macOS, where the previous
// proxy connection can otherwise leave the test/run port in TIME_WAIT and
// make the next fail-closed run refuse to start.
func listenBridge(network, address string) (net.Listener, error) {
	var config net.ListenConfig
	config.Control = func(_, _ string, conn syscall.RawConn) error {
		var socketErr error
		if err := conn.Control(func(fd uintptr) {
			socketErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
		}); err != nil {
			return err
		}
		return socketErr
	}
	return config.Listen(context.Background(), network, address)
}
