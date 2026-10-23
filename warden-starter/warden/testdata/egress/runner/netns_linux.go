//go:build linux

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const envInNS = "WARDEN_EGRESS_IN_NS"

// netnsSupported reports whether the Linux test-network prerequisites are
// present. It is never used to claim a PASS by itself.
func netnsSupported() (bool, string) {
	if _, err := exec.LookPath("unshare"); err != nil {
		return false, "unshare not found (util-linux)"
	}
	if _, err := exec.LookPath("ip"); err != nil {
		return false, "ip not found (iproute2)"
	}
	return true, ""
}

// reexecInNetns relaunches this runner inside an unprivileged user+net+mount
// namespace. The test collectors and Warden's host-side proxy live in that
// namespace; the sandbox backend gets its own fresh network namespace, so
// direct egress has no route and only the bridge path remains.
func reexecInNetns(rest []string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	args := []string{"--user", "--map-root-user", "--net", "--mount", "--propagation", "private", "--", self}
	args = append(args, rest...)
	cmd := exec.Command("unshare", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), envInNS+"=1")
	return cmd.Run()
}

type netnsState struct {
	IPv4       bool     `json:"ipv4"`
	IPv6       bool     `json:"ipv6"`
	Addresses  []string `json:"addresses"`
	Notes      []string `json:"notes,omitempty"`
	ResolvConf string   `json:"resolv_conf"`
	Hosts      string   `json:"hosts"`
}

// setupNetns configures loopback test addresses and controlled resolver
// files inside the namespace. Nothing outside the namespace is modified.
func setupNetns(tmpRoot, dnsIP string) (*netnsState, error) {
	st := &netnsState{}
	run := func(args ...string) (string, error) {
		out, err := exec.Command("ip", args...).CombinedOutput()
		return string(out), err
	}
	if _, err := run("link", "set", "lo", "up"); err != nil {
		return nil, fmt.Errorf("ip link set lo up: %w", err)
	}
	ipv4Addrs := []string{
		"192.0.2.1/32", "192.0.2.53/32", "192.0.2.66/32",
		"10.0.0.1/32", "169.254.169.254/32",
	}
	for _, cidr := range ipv4Addrs {
		out, err := run("addr", "add", cidr, "dev", "lo")
		if err != nil && !strings.Contains(out, "File exists") {
			return nil, fmt.Errorf("ip addr add %s: %v: %s", cidr, err, strings.TrimSpace(out))
		}
		st.Addresses = append(st.Addresses, cidr)
	}
	st.IPv4 = true
	ipv6Addrs := []string{"2001:db8::1/128", "2001:db8::2/128"}
	v6ok := true
	for _, cidr := range ipv6Addrs {
		out, err := run("-6", "addr", "add", cidr, "dev", "lo")
		if err != nil && !strings.Contains(out, "File exists") {
			v6ok = false
			st.Notes = append(st.Notes, fmt.Sprintf("ipv6 address %s unavailable: %s", cidr, strings.TrimSpace(out)))
			continue
		}
		st.Addresses = append(st.Addresses, cidr)
	}
	if v6ok {
		if conn, err := net.Listen("tcp", "[2001:db8::2]:0"); err == nil {
			conn.Close()
			st.IPv6 = true
		} else {
			st.Notes = append(st.Notes, "ipv6 loopback bind failed: "+err.Error())
		}
	}

	// Controlled resolver configuration: the host-side egress proxy is the
	// only process that resolves names, and it must resolve them from the
	// test DNS server.
	resolv := "nameserver " + dnsIP + "\noptions timeout:2 attempts:2 ndots:1\n"
	resolvPath := filepath.Join(tmpRoot, "resolv.conf")
	if err := os.WriteFile(resolvPath, []byte(resolv), 0o644); err != nil {
		return nil, err
	}
	if err := bindMount(resolvPath, "/etc/resolv.conf"); err != nil {
		return nil, fmt.Errorf("bind /etc/resolv.conf: %w", err)
	}
	st.ResolvConf = resolv
	hosts := "127.0.0.1 localhost\n::1 localhost\n"
	hostsPath := filepath.Join(tmpRoot, "hosts")
	if err := os.WriteFile(hostsPath, []byte(hosts), 0o644); err != nil {
		return nil, err
	}
	if err := bindMount(hostsPath, "/etc/hosts"); err != nil {
		st.Notes = append(st.Notes, "hosts bind failed (non-fatal): "+err.Error())
	}
	st.Hosts = hosts
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		st.Notes = append(st.Notes, "make-rprivate failed (non-fatal): "+err.Error())
	}
	return st, nil
}

func bindMount(src, dst string) error {
	if _, err := os.Stat(dst); err != nil {
		if os.IsNotExist(err) {
			if werr := os.WriteFile(dst, nil, 0o644); werr != nil {
				return werr
			}
		} else {
			return err
		}
	}
	return syscall.Mount(src, dst, "", syscall.MS_BIND, "")
}
