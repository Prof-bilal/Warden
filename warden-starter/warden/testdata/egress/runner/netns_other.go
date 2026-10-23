//go:build !linux

package main

import "errors"

const envInNS = "WARDEN_EGRESS_IN_NS"

func netnsSupported() (bool, string) {
	return false, "Linux test network namespaces are not available on this platform"
}

func reexecInNetns(rest []string) error {
	return errors.New("not supported")
}

type netnsState struct {
	IPv4       bool     `json:"ipv4"`
	IPv6       bool     `json:"ipv6"`
	Addresses  []string `json:"addresses"`
	Notes      []string `json:"notes,omitempty"`
	ResolvConf string   `json:"resolv_conf"`
	Hosts      string   `json:"hosts"`
}

func setupNetns(tmpRoot, dnsIP string) (*netnsState, error) {
	return &netnsState{Notes: []string{"host network mode: no test namespace"}}, nil
}
