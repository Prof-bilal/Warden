//go:build windows

package windows

import (
	"testing"
	"unsafe"
)

// These assertions pin the FWPM_* struct mirrors to the Windows SDK's x64
// layout. A mismatch is not cosmetic: BFE marshals FWPM_FILTER0 using its own
// definition, so a wrong offset or size makes FwpmFilterAdd0 fail with
// RPC_X_BAD_STUB_DATA (0x6f7) and the Windows backend refuses to start.
//
// If one of these ever fails, fix the struct in wfp.go to match the SDK again;
// do not adjust the expected number, which is the ABI.

func TestFWPMFilterLayout(t *testing.T) {
	if got := unsafe.Sizeof(fwpmFilter{}); got != 200 {
		t.Fatalf("sizeof(FWPM_FILTER0) = %d, want 200", got)
	}
	var f fwpmFilter
	for _, c := range []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"filterKey", unsafe.Offsetof(f.filterKey), 0},
		{"displayData", unsafe.Offsetof(f.displayData), 16},
		{"flags", unsafe.Offsetof(f.flags), 32},
		{"providerKey", unsafe.Offsetof(f.providerKey), 40},
		{"providerData", unsafe.Offsetof(f.providerData), 48},
		{"layerKey", unsafe.Offsetof(f.layerKey), 64},
		{"subLayerKey", unsafe.Offsetof(f.subLayerKey), 80},
		{"weight", unsafe.Offsetof(f.weight), 96},
		{"numFilterConditions", unsafe.Offsetof(f.numFilterConditions), 112},
		{"filterCondition", unsafe.Offsetof(f.filterCondition), 120},
		{"action", unsafe.Offsetof(f.action), 128},
		{"providerContext", unsafe.Offsetof(f.providerContext), 152},
		{"reserved", unsafe.Offsetof(f.reserved), 168},
		{"filterId", unsafe.Offsetof(f.filterId), 176},
		{"effectiveWeight", unsafe.Offsetof(f.effectiveWeight), 184},
	} {
		if c.got != c.want {
			t.Errorf("FWPM_FILTER0.%s offset = %d, want %d", c.name, c.got, c.want)
		}
	}
}

func TestFWPMSublayerLayout(t *testing.T) {
	if got := unsafe.Sizeof(fwpmSublayer{}); got != 72 {
		t.Fatalf("sizeof(FWPM_SUBLAYER0) = %d, want 72", got)
	}
	var s fwpmSublayer
	for _, c := range []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"subLayerKey", unsafe.Offsetof(s.subLayerKey), 0},
		{"displayData", unsafe.Offsetof(s.displayData), 16},
		{"flags", unsafe.Offsetof(s.flags), 32},
		{"providerKey", unsafe.Offsetof(s.providerKey), 40},
		{"providerData", unsafe.Offsetof(s.providerData), 48},
		{"weight", unsafe.Offsetof(s.weight), 64},
	} {
		if c.got != c.want {
			t.Errorf("FWPM_SUBLAYER0.%s offset = %d, want %d", c.name, c.got, c.want)
		}
	}
}

func TestFWPMNestedLayout(t *testing.T) {
	// FWP_CONDITION_VALUE0 keeps 8-byte alignment because its union holds a
	// 16-byte FWP_BYTE_ARRAY16. Without that alignment
	// FWPM_FILTER_CONDITION0 shrinks to 44 bytes and every condition after the
	// first is misread.
	if got := unsafe.Sizeof(fwpConditionValue{}); got != 24 {
		t.Errorf("sizeof(FWP_CONDITION_VALUE0) = %d, want 24", got)
	}
	if got := unsafe.Alignof(fwpConditionValue{}); got != 8 {
		t.Errorf("alignof(FWP_CONDITION_VALUE0) = %d, want 8", got)
	}
	if got := unsafe.Sizeof(fwpValue{}); got != 16 {
		t.Errorf("sizeof(FWP_VALUE0) = %d, want 16", got)
	}
	if got := unsafe.Sizeof(fwpmAction{}); got != 24 {
		t.Errorf("sizeof(FWPM_ACTION0) = %d, want 24", got)
	}
	if got := unsafe.Sizeof(fwpmFilterCondition{}); got != 48 {
		t.Errorf("sizeof(FWPM_FILTER_CONDITION0) = %d, want 48", got)
	}
	if got := unsafe.Sizeof(fwpByteBlob{}); got != 16 {
		t.Errorf("sizeof(FWP_BYTE_BLOB) = %d, want 16", got)
	}
	if got := unsafe.Sizeof(fwpmDisplayData{}); got != 16 {
		t.Errorf("sizeof(FWPM_DISPLAY_DATA0) = %d, want 16", got)
	}
}
