//go:build windows

package proxy

import "net"

func listenBridge(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}
