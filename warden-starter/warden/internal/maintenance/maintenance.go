package maintenance

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/warden-sandbox/warden/internal/clientconfig"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/privatefile"
)

type Change struct {
	Surface string   `json:"surface"`
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
}
type Diff struct {
	Changes        []Change `json:"changes"`
	CommandChanged bool     `json:"command_changed"`
	LimitsExpanded bool     `json:"limits_expanded"`
	ReviewRequired bool     `json:"review_required"`
}

func Compare(a, b policy.Policy) Diff {
	d := Diff{}
	for _, pair := range []struct {
		name string
		a, b []string
	}{{"read", a.Filesystem.Read, b.Filesystem.Read}, {"write", a.Filesystem.Write, b.Filesystem.Write}, {"network", a.Network.Allow, b.Network.Allow}, {"env_names", a.Env.Allow, b.Env.Allow}} {
		old, new := map[string]bool{}, map[string]bool{}
		for _, v := range pair.a {
			old[v] = true
		}
		for _, v := range pair.b {
			new[v] = true
		}
		c := Change{Surface: pair.name, Added: []string{}, Removed: []string{}}
		for v := range new {
			if !old[v] {
				c.Added = append(c.Added, v)
			}
		}
		for v := range old {
			if !new[v] {
				c.Removed = append(c.Removed, v)
			}
		}
		sort.Strings(c.Added)
		sort.Strings(c.Removed)
		d.Changes = append(d.Changes, c)
		if len(c.Added) > 0 {
			d.ReviewRequired = true
		}
	}
	ac, _ := json.Marshal(a.Command)
	bc, _ := json.Marshal(b.Command)
	d.CommandChanged = string(ac) != string(bc)
	d.LimitsExpanded = (a.Limits.MemoryMB > 0 && (b.Limits.MemoryMB == 0 || b.Limits.MemoryMB > a.Limits.MemoryMB)) || (a.Limits.TimeoutS > 0 && (b.Limits.TimeoutS == 0 || b.Limits.TimeoutS > a.Limits.TimeoutS))
	d.ReviewRequired = d.ReviewRequired || d.CommandChanged || d.LimitsExpanded
	// Existing MCP proxy policies may change authorization independently of
	// process grants. Mark every MCP change for review without printing bodies.
	am, _ := json.Marshal(a.MCP)
	bm, _ := json.Marshal(b.MCP)
	if string(am) != string(bm) {
		d.ReviewRequired = true
	}
	return d
}

type Record struct {
	Time    time.Time `json:"time"`
	Client  string    `json:"client"`
	Event   string    `json:"event"`
	Seconds int       `json:"seconds"`
}

var Events = []string{"setup-completed", "setup-failed", "first-protected-task", "repeat-use", "rollback", "badge-activation"}

func Validate(r Record) error {
	if _, e := clientconfig.Get(r.Client); e != nil {
		return e
	}
	ok := false
	for _, v := range Events {
		if v == r.Event {
			ok = true
		}
	}
	if !ok || r.Seconds < 0 || r.Seconds > 86400 || r.Time.IsZero() {
		return fmt.Errorf("invalid local report record")
	}
	return nil
}
func Read(path string) ([]Record, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 1024), 16384)
	var out []Record
	for s.Scan() {
		if len(out) >= 10000 {
			return nil, fmt.Errorf("local report record limit")
		}
		var r Record
		if json.Unmarshal(s.Bytes(), &r) != nil || Validate(r) != nil {
			return nil, fmt.Errorf("invalid local report")
		}
		out = append(out, r)
	}
	return out, s.Err()
}
func Append(path string, r Record) error {
	if e := Validate(r); e != nil {
		return e
	}
	if i, e := os.Lstat(path); e == nil {
		if !i.Mode().IsRegular() {
			return fmt.Errorf("report symlink refused")
		}
		if _, e = Read(path); e != nil {
			return e
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	f, e := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	defer f.Close()
	if e = privatefile.Protect(path, false); e != nil {
		return e
	}
	if e = json.NewEncoder(f).Encode(r); e != nil {
		return e
	}
	return f.Sync()
}

type Summary struct {
	Records              int            `json:"records"`
	Counts               map[string]int `json:"counts"`
	MedianSetupSeconds   float64        `json:"median_setup_seconds"`
	SetupFailureFraction float64        `json:"setup_failure_fraction"`
	Note                 string         `json:"note"`
}

func Summarize(rows []Record) Summary {
	s := Summary{Records: len(rows), Counts: map[string]int{}, Note: "Explicit local event counts; not unique users, automatic verification, or measured pilot results."}
	var durations []int
	for _, r := range rows {
		s.Counts[r.Event]++
		if r.Event == "setup-completed" {
			durations = append(durations, r.Seconds)
		}
	}
	sort.Ints(durations)
	if n := len(durations); n > 0 {
		if n%2 == 1 {
			s.MedianSetupSeconds = float64(durations[n/2])
		} else {
			s.MedianSetupSeconds = float64(durations[n/2-1]+durations[n/2]) / 2
		}
	}
	if n := s.Counts["setup-completed"] + s.Counts["setup-failed"]; n > 0 {
		s.SetupFailureFraction = float64(s.Counts["setup-failed"]) / float64(n)
	}
	return s
}
