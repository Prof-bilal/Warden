package policy

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// TemporaryGrant adds one exact host or read path until Expires.
type TemporaryGrant struct {
	Host    string    `yaml:"host,omitempty"`
	Path    string    `yaml:"path,omitempty"`
	Expires time.Time `yaml:"expires"`
}

func (p Policy) validateTemporaryGrants() error {
	if p.Version < 0 || p.Version > 2 {
		return fmt.Errorf("unsupported policy version %d", p.Version)
	}
	if len(p.TemporaryGrants) == 0 {
		return nil
	}
	if p.Version != 2 || p.Agent != nil || p.MCP != nil {
		return fmt.Errorf("temporary_grants require a version 2 process policy")
	}
	for i, grant := range p.TemporaryGrants {
		if grant.Expires.IsZero() || (grant.Host == "") == (grant.Path == "") {
			return fmt.Errorf("temporary_grants[%d] requires expires and exactly one host or path", i)
		}
		if grant.Host != "" {
			if err := validateHost(grant.Host); err != nil || strings.Contains(grant.Host, "*") {
				return fmt.Errorf("temporary_grants[%d] requires an exact host", i)
			}
		}
		if grant.Path != "" {
			if err := checkGrantPath(grant.Path, "read"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Policy) dropExpiredGrants(now time.Time) {
	p.TemporaryGrants = slices.DeleteFunc(p.TemporaryGrants, func(g TemporaryGrant) bool { return !g.Expires.After(now) })
}

// WithActiveGrants returns a runtime policy. Keep the original policy when
// saving edits so temporary permissions never enter the permanent lists.
func (p Policy) WithActiveGrants(now time.Time) Policy {
	if len(p.TemporaryGrants) == 0 {
		return p
	}
	p.Filesystem.Read = slices.Clone(p.Filesystem.Read)
	p.Network.Allow = slices.Clone(p.Network.Allow)
	for _, grant := range p.TemporaryGrants {
		if !grant.Expires.After(now) {
			continue
		}
		if grant.Host != "" {
			host := strings.ToLower(grant.Host)
			if slices.Contains(p.Network.Allow, host) {
				continue
			}
			p.Network.Allow = append(p.Network.Allow, host)
		} else {
			if CoversFile(p, grant.Path) {
				continue
			}
			p.Filesystem.Read = append(p.Filesystem.Read, grant.Path)
		}
		if p.Limits.GrantDeadline.IsZero() || grant.Expires.Before(p.Limits.GrantDeadline) {
			p.Limits.GrantDeadline = grant.Expires
		}
	}
	return p
}

// WallClockLimit returns the effective time left and the reason for the limit.
// A zero deadline with an empty reason denotes an unlimited run.
func (l Limits) WallClockLimit(now time.Time) (time.Duration, string) {
	timeout := time.Duration(l.TimeoutS) * time.Second
	if !l.GrantDeadline.IsZero() {
		remain := l.GrantDeadline.Sub(now)
		if timeout == 0 || remain <= timeout {
			return remain, "temporary grant expired"
		}
	}
	if timeout > 0 {
		return timeout, "wall-clock timeout"
	}
	return 0, ""
}
