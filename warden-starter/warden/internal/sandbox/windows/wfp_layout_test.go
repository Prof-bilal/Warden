//go:build windows && amd64

package windows

import (
	"testing"
	"unsafe"
)

// TestWFPStructLayout pins the WFP mirrors to the amd64 offsets of
// fwpmtypes.h; a drift makes FwpmFilterAdd0 fail with RPC_X_BAD_STUB_DATA.
func TestWFPStructLayout(t *testing.T) {
	var f fwpmFilter
	var s fwpmSublayer
	var c fwpmFilterCondition
	for _, x := range []struct {
		name      string
		got, want uintptr
	}{
		{"sizeof FWP_VALUE0", unsafe.Sizeof(fwpValue{}), 16},
		{"sizeof FWP_CONDITION_VALUE0", unsafe.Sizeof(fwpConditionValue{}), 16},
		{"sizeof FWPM_FILTER_CONDITION0", unsafe.Sizeof(c), 40},
		{"FILTER0.displayData", unsafe.Offsetof(f.displayData), 16},
		{"FILTER0.flags", unsafe.Offsetof(f.flags), 32},
		{"FILTER0.providerKey", unsafe.Offsetof(f.providerKey), 40},
		{"FILTER0.providerData", unsafe.Offsetof(f.providerData), 48},
		{"FILTER0.layerKey", unsafe.Offsetof(f.layerKey), 64},
		{"FILTER0.subLayerKey", unsafe.Offsetof(f.subLayerKey), 80},
		{"FILTER0.weight", unsafe.Offsetof(f.weight), 96},
		{"FILTER0.numFilterConditions", unsafe.Offsetof(f.numFilterConditions), 112},
		{"FILTER0.filterCondition", unsafe.Offsetof(f.filterCondition), 120},
		{"FILTER0.action", unsafe.Offsetof(f.action), 128},
		{"FILTER0.providerContext", unsafe.Offsetof(f.providerContext), 152},
		{"FILTER0.reserved", unsafe.Offsetof(f.reserved), 168},
		{"FILTER0.filterId", unsafe.Offsetof(f.filterID), 176},
		{"FILTER0.effectiveWeight", unsafe.Offsetof(f.effectiveWeight), 184},
		{"sizeof FWPM_FILTER0", unsafe.Sizeof(f), 200},
		{"SESSION0.flags", unsafe.Offsetof(fwpmSession{}.flags), 32},
		{"SESSION0.processId", unsafe.Offsetof(fwpmSession{}.processID), 40},
		{"SESSION0.sid", unsafe.Offsetof(fwpmSession{}.sid), 48},
		{"SESSION0.kernelMode", unsafe.Offsetof(fwpmSession{}.kernelMode), 64},
		{"sizeof FWPM_SESSION0", unsafe.Sizeof(fwpmSession{}), 72},
		{"SUBLAYER0.flags", unsafe.Offsetof(s.flags), 32},
		{"SUBLAYER0.providerKey", unsafe.Offsetof(s.providerKey), 40},
		{"SUBLAYER0.providerData", unsafe.Offsetof(s.providerData), 48},
		{"SUBLAYER0.weight", unsafe.Offsetof(s.weight), 64},
		{"sizeof FWPM_SUBLAYER0", unsafe.Sizeof(s), 72},
	} {
		if x.got != x.want {
			t.Errorf("%s = %d, want %d", x.name, x.got, x.want)
		}
	}
}

// TestWFPGUIDsParsed guards against the zero-GUID regression: every layer and
// condition key must be a real, distinct GUID.
func TestWFPGUIDsParsed(t *testing.T) {
	seen := map[string]bool{}
	for _, g := range []struct {
		name string
		s    string
	}{
		{"ALE_AUTH_CONNECT_V4", guidLayerALEAuthConnectV4.String()},
		{"ALE_AUTH_CONNECT_V6", guidLayerALEAuthConnectV6.String()},
		{"ALE_PACKAGE_ID", guidConditionALEPackageID.String()},
		{"FLAGS", guidConditionFlags.String()},
		{"IP_REMOTE_PORT", guidConditionIPRemotePort.String()},
	} {
		if g.s == "{00000000-0000-0000-0000-000000000000}" || seen[g.s] {
			t.Errorf("%s GUID is zero or duplicated: %s", g.name, g.s)
		}
		seen[g.s] = true
	}
}

// TestETWLogfileLayout pins EVENT_TRACE_LOGFILEW to its x64 offsets.
func TestETWLogfileLayout(t *testing.T) {
	var lf eventTraceLogfile
	for _, x := range []struct {
		name      string
		got, want uintptr
	}{
		{"LoggerName", unsafe.Offsetof(lf.loggerName), 8},
		{"ProcessTraceMode", unsafe.Offsetof(lf.processTraceMode), 28},
		{"CurrentEvent", unsafe.Offsetof(lf.currentEvent), 32},
		{"LogfileHeader", unsafe.Offsetof(lf.logfileHeader), 120},
		{"BufferCallback", unsafe.Offsetof(lf.bufferCallback), 400},
		{"EventRecordCallback", unsafe.Offsetof(lf.eventRecordCallback), 424},
		{"Context", unsafe.Offsetof(lf.context), 440},
		{"sizeof", unsafe.Sizeof(lf), 448},
	} {
		if x.got != x.want {
			t.Errorf("EVENT_TRACE_LOGFILEW.%s = %d, want %d", x.name, x.got, x.want)
		}
	}
}

// TestETWEventRecordLayout pins EVENT_RECORD (evntcons.h) to its x64 offsets.
func TestETWEventRecordLayout(t *testing.T) {
	var r eventRecord
	for _, x := range []struct {
		name      string
		got, want uintptr
	}{
		{"sizeof EVENT_HEADER", unsafe.Sizeof(r.header), 80},
		{"EventDescriptor", unsafe.Offsetof(r.header.descriptor), 40},
		{"UserDataLength", unsafe.Offsetof(r.userDataLength), 86},
		{"ExtendedData", unsafe.Offsetof(r.extendedData), 88},
		{"UserData", unsafe.Offsetof(r.userData), 96},
		{"sizeof EVENT_RECORD", unsafe.Sizeof(r), 112},
	} {
		if x.got != x.want {
			t.Errorf("EVENT_RECORD.%s = %d, want %d", x.name, x.got, x.want)
		}
	}
}
