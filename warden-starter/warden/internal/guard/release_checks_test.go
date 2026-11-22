package guard

import (
	"testing"
	"time"
)

func TestDecideRelease(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	older := ReleaseFacts{Name: "@example/tool", Version: "1.0.3", PublishedAt: now.Add(-48 * time.Hour)}
	cases := []struct {
		name     string
		edit     func(*ReleaseFacts, *ReleaseRules)
		blocks   int
		warnings int
	}{
		{name: "passes at the age threshold"},
		{
			name:   "fresh release",
			edit:   func(f *ReleaseFacts, _ *ReleaseRules) { f.PublishedAt = now.Add(-2 * time.Hour) },
			blocks: 1,
		},
		{
			name: "explicit age override still blocks scripts",
			edit: func(f *ReleaseFacts, r *ReleaseRules) {
				f.PublishedAt = now.Add(-2 * time.Hour)
				f.InstallScripts = []string{"postinstall"}
				r.OverrideAge = true
			},
			blocks: 1, warnings: 1,
		},
		{
			name:     "missing publication time",
			edit:     func(f *ReleaseFacts, _ *ReleaseRules) { f.PublishedAt = time.Time{} },
			warnings: 1,
		},
		{
			name: "strict missing publication time",
			edit: func(f *ReleaseFacts, r *ReleaseRules) {
				f.PublishedAt = time.Time{}
				r.StrictMetadata = true
			},
			blocks: 1,
		},
		{
			name: "binding gyp with wildcard allowlist",
			edit: func(f *ReleaseFacts, r *ReleaseRules) {
				f.HasBindingGYP = true
				r.ScriptAllowlist = []string{"@example/tool@*"}
			},
			blocks: 1,
		},
		{
			name: "binding gyp with exact allowlist",
			edit: func(f *ReleaseFacts, r *ReleaseRules) {
				f.HasBindingGYP = true
				r.ScriptAllowlist = []string{"@example/tool@1.0.3"}
			},
		},
		{
			name:   "version range",
			edit:   func(f *ReleaseFacts, _ *ReleaseRules) { f.Version = "^1.0.3" },
			blocks: 1,
		},
		{
			name: "changed maintainers",
			edit: func(f *ReleaseFacts, _ *ReleaseRules) {
				f.PreviousMaintainers = []string{"first"}
				f.Maintainers = []string{"second"}
			},
			warnings: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts, rules := older, ReleaseRules{}
			if tc.edit != nil {
				tc.edit(&facts, &rules)
			}
			got := DecideRelease(now, facts, rules)
			if len(got.Blocks) != tc.blocks || len(got.Warnings) != tc.warnings {
				t.Fatalf("got %+v, want %d blocks and %d warnings", got, tc.blocks, tc.warnings)
			}
		})
	}
}
