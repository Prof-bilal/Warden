package maintenance

import (
	"github.com/warden-sandbox/warden/internal/policy"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPermissionReviewAndLocalReport(t *testing.T) {
	a := policy.Policy{Filesystem: policy.Filesystem{Read: []string{"/selected"}}, Limits: policy.Limits{MemoryMB: 512, TimeoutS: 30}}
	b := a
	b.Filesystem.Write = []string{"/selected"}
	b.Env.Allow = []string{"API_TOKEN"}
	b.Limits.TimeoutS = 0
	d := Compare(a, b)
	if !d.ReviewRequired || !d.LimitsExpanded || len(d.Changes[1].Added) != 1 {
		t.Fatal("permission expansion not marked")
	}
	if Compare(a, a).ReviewRequired {
		t.Fatal("unchanged policy requires review")
	}
	path := filepath.Join(t.TempDir(), "pilot.jsonl")
	for _, r := range []Record{{time.Now().UTC(), "cursor", "setup-completed", 60}, {time.Now().UTC(), "cursor", "setup-completed", 120}, {time.Now().UTC(), "cursor", "setup-failed", 0}} {
		if e := Append(path, r); e != nil {
			t.Fatal(e)
		}
	}
	rows, e := Read(path)
	if e != nil {
		t.Fatal(e)
	}
	s := Summarize(rows)
	if s.MedianSetupSeconds != 90 || s.Records != 3 || s.SetupFailureFraction != float64(1)/3 {
		t.Fatalf("wrong report: %+v", s)
	}
	if Append(path, Record{time.Now(), "cursor", "tool-body-with-token", 0}) == nil {
		t.Fatal("arbitrary payload recorded")
	}
	_ = os.WriteFile(path, []byte("bad JSON\n"), 0600)
	if _, e := Read(path); e == nil {
		t.Fatal("malformed report ignored")
	}
}
