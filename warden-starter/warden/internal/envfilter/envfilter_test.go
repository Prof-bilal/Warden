package envfilter

import "testing"

func TestFilterAllowsOnlyListed(t *testing.T) {
	parent := []string{"HOME=/home/u", "GITHUB_TOKEN=secret", "PATH=/bin"}
	got := Filter(parent, []string{"GITHUB_TOKEN"})
	want := []string{"GITHUB_TOKEN=secret"}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("Filter = %v, want %v", got, want)
	}
}

func TestFilterEmptyAllowlist(t *testing.T) {
	parent := []string{"HOME=/home/u", "PATH=/bin"}
	if got := Filter(parent, nil); len(got) != 0 {
		t.Errorf("Filter with empty allowlist = %v, want empty", got)
	}
}

func TestFilterKeepsLastDuplicate(t *testing.T) {
	parent := []string{"A=one", "A=two", "B=three"}
	got := Filter(parent, []string{"A"})
	want := []string{"A=two"}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("Filter = %v, want %v", got, want)
	}
}

func TestFilterDropsMisshapenEntries(t *testing.T) {
	parent := []string{"MALFORMED", "A=ok"}
	got := Filter(parent, []string{"A", "MALFORMED"})
	want := []string{"A=ok"}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("Filter = %v, want %v", got, want)
	}
}
