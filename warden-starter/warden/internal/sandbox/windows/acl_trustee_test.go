//go:build windows

package windows

import "testing"

// Regression: TRUSTEE_IS_SID must be 0. Using 3 (TRUSTEE_IS_OBJECTS_AND_SID)
// caused SetEntriesInAcl to return ERROR_INVALID_PARAMETER on every
// filesystem grant, which surfaced on Windows CI as:
//
//	grant filesystem access to C:\Users\RUNNER~1\...
//	SetEntriesInAcl: The parameter is incorrect.
func TestTrusteeIsSidConstant(t *testing.T) {
	if trusteeIsSid != 0 {
		t.Fatalf("trusteeIsSid = %d, want 0 (TRUSTEE_IS_SID); 3 is TRUSTEE_IS_OBJECTS_AND_SID", trusteeIsSid)
	}
}

func TestLongPathNoTildeUnchanged(t *testing.T) {
	in := `C:\Users\runner\AppData\Local\Temp\x`
	if got := longPath(in); got != in {
		t.Fatalf("longPath(%q) = %q, want unchanged", in, got)
	}
}
