package policy

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
)

func TestVerifiedPolicyRejectsPermissionExpansion(t *testing.T) {
	original := "filesystem: {}\n"
	path := writeTempPolicy(t, original)
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(original)))
	if _, e := LoadVerified(path, digest); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(path, []byte("filesystem: {read: ['/private-data']}\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := LoadVerified(path, digest); e == nil {
		t.Fatal("changed permissions accepted by pinned wrapper")
	}
	if _, e := Load(path); e != nil {
		t.Fatal("normal explicit policy editing broken")
	}
}
