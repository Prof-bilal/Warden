package proxy

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// ParseBridgeArgs parses __proxy-bridge command-line arguments
// (--socket PATH --listen ADDR -- target...) using the same rules as the
// warden binary's cmdProxyBridge. It exists so the in-process bridge (used
// by the darwin integration tests via TestMain) and the CLI entry point
// cannot drift apart.
func ParseBridgeArgs(args []string) (socket, listen string, target []string, err error) {
	i := 0
	for i < len(args) && args[i] != "--" {
		if i+1 >= len(args) {
			return "", "", nil, fmt.Errorf("flag requires a value")
		}
		switch args[i] {
		case "--socket":
			socket = args[i+1]
		case "--listen":
			listen = args[i+1]
		default:
			return "", "", nil, fmt.Errorf("unknown flag %q", args[i])
		}
		i += 2
	}
	if socket == "" || listen == "" || i == len(args) || i+1 == len(args) {
		return "", "", nil, fmt.Errorf("missing socket, listen address, or command")
	}
	return socket, listen, args[i+1:], nil
}

// RunBridge runs inside the sandbox network namespace.  It exposes a
// loopback-only HTTP proxy endpoint and forwards every accepted connection to
// the host-side policy proxy over its bind-mounted Unix socket.  The target
// has no non-loopback network interface, so it cannot bypass this bridge.
func RunBridge(socketPath, listenAddr string, command []string) (int, error) {
	if len(command) == 0 {
		return 0, fmt.Errorf("proxy bridge: no target command")
	}
	l, err := net.Listen("tcp4", listenAddr)
	if err != nil {
		return 0, fmt.Errorf("proxy bridge: listen on %s: %w", listenAddr, err)
	}
	defer l.Close()

	cmd := exec.Command(command[0], command[1:]...)
	// Leaving these nil would attach /dev/null.  The bridge is executed by
	// Warden, so inherit its stdio explicitly via the process descriptors.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("proxy bridge: start target: %w", err)
	}

	stop := make(chan struct{})
	go func() {
		defer close(stop)
		for {
			client, err := l.Accept()
			if err != nil {
				return
			}
			go forward(client, socketPath)
		}
	}()

	// Make Ctrl-C and normal termination behave like executing the target
	// directly, rather than leaving it behind as the bridge exits.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		_ = l.Close()
		<-stop
		if err == nil {
			return 0, nil
		}
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 0, fmt.Errorf("proxy bridge: wait for target: %w", err)
	case sig := <-signals:
		_ = cmd.Process.Signal(sig)
		err := <-done
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 0, err
	}
}

// forward proxies one accepted bridge connection to the upstream Unix
// socket. On Windows there is no Unix-socket story for AppContainer
// processes; the Windows backend dials the host-side proxy over loopback
// TCP instead, which its WFP filters hard-permit.
func forward(client net.Conn, socketPath string) {
	defer client.Close()
	upstream, err := net.Dial("unix", socketPath)
	if err != nil {
		return
	}
	defer upstream.Close()
	pump(client, upstream)
}

// pump copies both directions between two connections until one side closes.
func pump(a, b net.Conn) {
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(a, b); done <- struct{}{} }()
	go func() { _, _ = io.Copy(b, a); done <- struct{}{} }()
	<-done
}
