package packs

import "testing"

func TestFindByArtifact(t *testing.T) {
	original := catalog
	t.Cleanup(func() { catalog = original })
	catalog = []byte(`[
		{"id":"candidate","artifact":"@example/tool","version":"1.0.0","status":"candidate"},
		{"id":"reviewed","artifact":"@example/tool","version":"1.0.1","status":"reviewed"}
	]`)

	pack, ok := FindByArtifact("@example/tool", "1.0.1")
	if !ok || pack.ID != "reviewed" {
		t.Fatalf("exact reviewed match = %q, %t", pack.ID, ok)
	}
	for _, version := range []string{"", "1.0.0", "1.0.2", "^1.0.1"} {
		if _, ok := FindByArtifact("@example/tool", version); ok {
			t.Errorf("unexpected match for version %q", version)
		}
	}
	if _, ok := FindByArtifact("@example/other", "1.0.1"); ok {
		t.Error("unexpected match for a different artifact")
	}
}
