//go:build !windows

package proxy

import (
	"context"
	"errors"
	"net"
	"syscall"
	"time"
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
	var lastErr error
	for attempt := 0; attempt < 40; attempt++ {
		listener, err := config.Listen(context.Background(), network, address)
		if err == nil {
			return listener, nil
		}
		lastErr = err
		// A terminated Seatbelt child can take a short moment to release the
		// fixed bridge port. Retry only the transient bind errors; all other
		// failures remain fail-closed and return immediately.
		if !errors.Is(err, syscall.EADDRINUSE) && !errors.Is(err, syscall.EPERM) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	return nil, lastErr
}
