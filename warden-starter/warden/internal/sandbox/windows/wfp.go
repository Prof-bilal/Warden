//go:build windows

package windows

import (
	"crypto/sha256"
	"fmt"
	"runtime"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// This file implements the Windows Filtering Platform (WFP) egress filters: a
// dedicated sublayer that hard-permits only the loopback proxy bridge and
// hard-blocks every other outbound IPv4/IPv6 connection from the sandboxed
// image. The AppContainer token itself denies all outbound capability, so these
// filters are belt-and-suspenders and the mechanism that re-opens the single
// loopback path the egress proxy needs.
//
// All GUIDs and struct mirrors below come from fwpmu.h / fwptypes.h / winnt.h.
// They are constants that a WFP implementation cannot invent; the escape tests
// on a Windows runner pin them in place.

// FWPM_ACTION_* (fwpmu.h).
const (
	fwpmActionBlock  = 0x00001001 // FWP_ACTION_BLOCK (0x1 | FWP_ACTION_FLAG_TERMINATING)
	fwpmActionPermit = 0x00001002 // FWP_ACTION_PERMIT (0x2 | FWP_ACTION_FLAG_TERMINATING)
	fwpmActionNone   = 0x00000007 // FWP_ACTION_NONE
)

// FWP_MATCH_TYPE (fwptypes.h).
const (
	fwpmMatchEqual       = 0
	fwpmMatchFlagsAllSet = 6
)

// fwpmFilterFlagClearActionRight lets the hard permit win arbitration
// against lower-weight sublayers.
const fwpmFilterFlagClearActionRight = 0x00000008

// FWP_DATA_TYPE (fwptypes.h).
const (
	fwpUint8        = 0x01
	fwpUint16       = 0x02
	fwpUint32       = 0x03
	fwpByteArray16  = 0x0B
	fwpByteBlobType = 0x0C
	fwpSid          = 0x0D
)

// FWPM Condition field GUIDs (fwpmu.h). FWP_CONDITION_FLAG_IS_LOOPBACK marks
// traffic that stays on a loopback interface.
const fwpConditionFlagIsLoopback = 0x00000001

var (
	// FWPM_CONDITION_ALE_PACKAGE_ID: the AppContainer package SID, so the
	// filters match only this run's container, never other host processes.
	guidConditionALEPackageID = guidOf("71bc78fa-f17c-4997-a602-6abb261f351c")
	// FWPM_CONDITION_FLAGS carries filter flags such as IS_LOOPBACK.
	guidConditionFlags = guidOf("632ce23b-5167-435c-86d7-e903684aa80c")
	// FWPM_CONDITION_IP_REMOTE_PORT is the destination port condition.
	guidConditionIPRemotePort = guidOf("c35a604d-d22b-4e1a-91b4-68f674ee674b")
)

// Well-known WFP layers we attach to (fwpmu.h).
var (
	// FWPM_LAYER_ALE_AUTH_CONNECT_V4
	guidLayerALEAuthConnectV4 = guidOf("c38d57d1-05a7-4c33-904f-7fbceee60e82")
	// FWPM_LAYER_ALE_AUTH_CONNECT_V6
	guidLayerALEAuthConnectV6 = guidOf("4a72393b-319f-44bc-84c3-ba54dcb3b6b4")
)

// fwpByteBlob mirrors FWP_BYTE_BLOB (fwptypes.h).
type fwpByteBlob struct {
	size uint32
	data *byte
}

// fwpmDisplayData mirrors FWPM_DISPLAY_DATA (fwpmu.h).
type fwpmDisplayData struct {
	name        *uint16
	description *uint16
}

// The mirrors below follow fwptypes.h / fwpmtypes.h for amd64 exactly; their
// offsets are pinned by TestWFPStructLayout. A wrong layout makes
// FwpmFilterAdd0 fail with RPC_X_BAD_STUB_DATA (0x6f7).

// fwpConditionValue mirrors FWP_CONDITION_VALUE0: type + pointer-sized union.
type fwpConditionValue struct {
	valueType uint32
	_         uint32
	value     uint64 // union
}

func (v *fwpConditionValue) setUint32(x uint32) {
	v.valueType = fwpUint32
	*(*uint32)(unsafe.Pointer(&v.value)) = x
}

func (v *fwpConditionValue) setUint16(x uint16) {
	v.valueType = fwpUint16
	*(*uint16)(unsafe.Pointer(&v.value)) = x
}

func (v *fwpConditionValue) setSID(sid *windows.SID) {
	v.valueType = fwpSid
	*(*uintptr)(unsafe.Pointer(&v.value)) = uintptr(unsafe.Pointer(sid))
}

// fwpValue mirrors FWP_VALUE0 (same shape as the condition value).
type fwpValue struct {
	valueType uint32
	_         uint32
	value     uint64 // union
}

const fwpEmpty = 0

// fwpmFilterCondition mirrors FWPM_FILTER_CONDITION0.
type fwpmFilterCondition struct {
	fieldKey       windows.GUID
	matchType      uint32
	conditionValue fwpConditionValue
}

// fwpmAction mirrors FWPM_ACTION0: type + GUID union (filterType/calloutKey).
type fwpmAction struct {
	actionType uint32
	filterType windows.GUID
}

// fwpmFilter mirrors FWPM_FILTER0.
type fwpmFilter struct {
	filterKey           windows.GUID
	displayData         fwpmDisplayData
	flags               uint32
	providerKey         *windows.GUID
	providerData        fwpByteBlob
	layerKey            windows.GUID
	subLayerKey         windows.GUID
	weight              fwpValue
	numFilterConditions uint32
	filterCondition     *fwpmFilterCondition
	action              fwpmAction
	providerContext     struct{ lo, hi uint64 } // union rawContext/providerContextKey
	reserved            *windows.GUID
	filterID            uint64
	effectiveWeight     fwpValue
}

// fwpmSublayer mirrors FWPM_SUBLAYER0.
type fwpmSublayer struct {
	subLayerKey  windows.GUID
	displayData  fwpmDisplayData
	flags        uint32
	providerKey  *windows.GUID
	providerData fwpByteBlob
	weight       uint16
}

// guidOf parses a registry-style GUID string into a windows.GUID. These are
// compile-time constants; a failure indicates an SDL regression, returned as
// nil GUID and caught by the layer checks in installWFPEgress.
func guidOf(s string) windows.GUID {
	// GUIDFromString only accepts the braced form; a parse failure here is a
	// typo in a constant, never something to paper over with a zero GUID.
	g, err := windows.GUIDFromString("{" + s + "}")
	if err != nil {
		panic(fmt.Sprintf("windows: bad GUID constant %q: %v", s, err))
	}
	return g
}

// fwpmResultError turns a non-zero FWPM error code into a Go error.
func fwpmResultError(call string, code uintptr) error {
	if code == 0 {
		return nil
	}
	// FWPM_E_* codes are HRESULT-style (0x8032xxxx); encode them readably.
	return fmt.Errorf("WFP %s: code %#x", call, code)
}

// wfpSupported checks that all WFP procs required by the egress layer are
// resolvable in the DLL initWFP() selected. This is a fail-closed gate: if
// initWFP() couldn't find any host DLL for the API on this host, or any
// proc lookup fails (wrong DLL, wrong name, stripped Windows image), the
// sandbox refuses to start rather than panicking at Call time.
func wfpSupported() error {
	// initWFP() must have bound the procs; if none of the candidate DLLs
	// loaded on this host, every proc pointer is still nil. Report this as a
	// clear actionable error rather than letting a nil deref panic later.
	if procFwpmEngineOpen == nil {
		return failClose("WFP engine", errWFPDLLMissing)
	}
	for _, p := range []*windows.LazyProc{
		procFwpmEngineOpen,
		procFwpmEngineClose,
		procFwpmTransactionBegin,
		procFwpmTransactionCommit,
		procFwpmTransactionAbort,
		procFwpmSublayerAdd,
		procFwpmFilterAdd,
		procFwpmFreeMemory,
		procFwpmFilterDeleteById,
	} {
		if err := p.Find(); err != nil {
			return failClose("WFP engine", fmt.Errorf("required WFP procedure %q not found in %s: %w", p.Name, wfpDLLName(), err))
		}
	}
	// Export presence is not capability. GitHub-hosted Windows runners ship
	// fwpuclnt.dll but refuse FwpmEngineOpen0 with ERROR_NOT_SUPPORTED (50)
	// for every authnService, elevated or not, even from the SYSTEM account,
	// and the same refusal hits netsh/Get-NetFirewallProfile (diagnosed in
	// the ci.yml wfp-engine-probe rounds). Deeper hosted images still accept
	// the engine open but fail the first filter write with RPC_X_BAD_STUB_DATA
	// (0x6f7) on FwpmFilterAdd. Run the install path's exact sequence against
	// a throwaway sublayer and abort the transaction so nothing persists:
	// Supported() then reports the host's true ability to enforce egress
	// rules, and callers skip instead of failing on a refuse-to-run sandbox.
	engine, err := fwpmOpen()
	if err != nil {
		return err
	}
	defer func() { _, _, _ = procFwpmEngineClose.Call(uintptr(engine)) }()
	if r, _, _ := procFwpmTransactionBegin.Call(uintptr(engine), 0); r != 0 {
		return failClose("WFP engine", fwpmResultError("FwpmTransactionBegin", r))
	}
	subLayer := makeSubLayerGUID("warden-capability-probe")
	sub := fwpmSublayer{subLayerKey: subLayer, displayData: fwpmDisplayData{name: utf16Ptr("warden capability probe")}}
	if r, _, _ := procFwpmSublayerAdd.Call(uintptr(engine), uintptr(unsafe.Pointer(&sub)), 0); r != 0 {
		_ = fwpmAbort(engine)
		return failClose("WFP engine", fwpmResultError("FwpmSublayerAdd", r))
	}
	var probeIDs []uint64
	probeSID, _ := windows.StringToSid(appContainerSID("warden-capability-probe"))
	if err := addBlockAll(engine, subLayer, guidLayerALEAuthConnectV4, probeSID, &probeIDs); err != nil {
		_ = fwpmAbort(engine)
		return err
	}
	// Abort rolls the probe back: no filter, no sublayer, nothing persists.
	if err := fwpmAbort(engine); err != nil {
		return failClose("WFP engine", err)
	}
	return nil
}

// rpcAuthnWinNT is RPC_C_AUTHN_WINNT (rpcdce.h): the NTLM SSP. FwpmEngineOpen0
// accepts only RPC_C_AUTHN_WINNT or RPC_C_AUTHN_DEFAULT for authnService, even
// for a local (NULL serverName) session. Passing RPC_C_AUTHN_NONE (0) is
// rejected with ERROR_NOT_SUPPORTED (50), which reads like "this host has no
// WFP" and fail-closes a perfectly healthy engine.
const rpcAuthnWinNT = 10

// fwpmSessionFlagDynamic is FWPM_SESSION_FLAG_DYNAMIC (fwpmtypes.h): every
// object added through the session is deleted when the session ends.
const fwpmSessionFlagDynamic = 0x00000001

// fwpmSession mirrors FWPM_SESSION0 (fwpmtypes.h), 72 bytes on x64.
type fwpmSession struct {
	sessionKey           windows.GUID
	displayData          fwpmDisplayData
	flags                uint32
	txnWaitTimeoutInMSec uint32
	processID            uint32
	sid                  *windows.SID
	username             *uint16
	kernelMode           int32
}

// fwpmOpen opens a handle to the filter engine in a dynamic session, so the
// run's sublayer and filters are removed by Windows when the handle closes
// or the process dies. A non-dynamic session left them installed until the
// Base Filtering Engine restarted (the sublayer on every run, the filters
// too whenever warden crashed or was killed).
func fwpmOpen() (windows.Handle, error) {
	var engine windows.Handle
	session := fwpmSession{flags: fwpmSessionFlagDynamic, displayData: fwpmDisplayData{name: utf16Ptr("warden")}}
	r, _, _ := procFwpmEngineOpen.Call(
		0,             // serverName: NULL, local engine
		rpcAuthnWinNT, // authnService: RPC_C_AUTHN_WINNT (allowed values are WINNT and DEFAULT)
		0,             // authIdentity: NULL, use the calling thread's credentials
		uintptr(unsafe.Pointer(&session)),
		uintptr(unsafe.Pointer(&engine)),
	)
	runtime.KeepAlive(&session)
	if err := fwpmResultError("FwpmEngineOpen", r); err != nil {
		return 0, failClose("open WFP engine", err)
	}
	return engine, nil
}

// makeSubLayerGUID derives a per-run sublayer GUID from the package SID so each
// sandbox isolates its filters without colliding with another run.
func makeSubLayerGUID(seed string) windows.GUID {
	raw := sha256Sum318(seed)
	var g windows.GUID
	copy(g.Data4[:], raw[8:16])
	return windows.GUID{Data1: binaryLE32(raw[0:4]), Data2: uint16(raw[4])<<8 | uint16(raw[5]), Data3: uint16(raw[6])<<8 | uint16(raw[7]), Data4: g.Data4}
}

// wfpSession owns the filter-engine handle, the per-run sublayer, and the
// buffers (image path byte blob) the installed filters reference so Go never
// moves memory under the engine.
type wfpSession struct {
	engine    windows.Handle
	subLayer  windows.GUID
	filterIDs []uint64
	pkg       *windows.SID // keeps the condition SID alive for the session
}

// installWFPEgress installs the WFP filters for one run. It fails closed: any
// error aborts the transaction and closes the engine so a partially configured
// policy can never leave a default-permit path.
func installWFPEgress(packageSID string, allowPort uint16, seedGUID string) (sess *wfpSession, err error) {
	if err := wfpSupported(); err != nil {
		return nil, err
	}
	engine, err := fwpmOpen()
	if err != nil {
		return nil, err
	}
	sess = &wfpSession{engine: engine, subLayer: makeSubLayerGUID(seedGUID)}
	// Tear down the engine only if we fail before committing; a committed
	// session is returned intact for the caller to release.
	defer func() {
		if err != nil {
			sess.close()
		}
	}()

	pkg, perr := windows.StringToSid(packageSID)
	if perr != nil {
		return nil, failClose("WFP package SID", perr)
	}
	sess.pkg = pkg

	if r, _, _ := procFwpmTransactionBegin.Call(uintptr(engine), 0); r != 0 {
		return nil, fwpmResultError("FwpmTransactionBegin", r)
	}

	// Top sublayer weight: sublayers are arbitrated high to low, so the hard
	// permit for the proxy port is decided before the built-in AppContainer
	// loopback isolation, and the deny-all is decided before anything that
	// might permit the container elsewhere.
	sub := fwpmSublayer{subLayerKey: sess.subLayer, displayData: fwpmDisplayData{name: utf16Ptr("warden sublayer")}, weight: 0xFFFF}
	if r, _, _ := procFwpmSublayerAdd.Call(uintptr(engine), uintptr(unsafe.Pointer(&sub)), 0); r != 0 {
		_ = fwpmAbort(engine)
		return nil, fwpmResultError("FwpmSublayerAdd", r)
	}

	for _, layer := range [2]windows.GUID{guidLayerALEAuthConnectV4, guidLayerALEAuthConnectV6} {
		// Hard-permit: the loopback proxy bridge from this image.
		if err := addPermitLoopback(engine, sess.subLayer, layer, allowPort, sess.pkg, &sess.filterIDs); err != nil {
			_ = fwpmAbort(engine)
			return nil, err
		}
		// Hard-block every other outbound connection for the image. UDP/DNS is
		// denied by the AppContainer token; this closes the TCP path.
		if err := addBlockAll(engine, sess.subLayer, layer, sess.pkg, &sess.filterIDs); err != nil {
			_ = fwpmAbort(engine)
			return nil, err
		}
	}

	if r, _, _ := procFwpmTransactionCommit.Call(uintptr(engine)); r != 0 {
		_ = fwpmAbort(engine)
		return nil, fwpmResultError("FwpmTransactionCommit", r)
	}
	return sess, nil
}

func fwpmAbort(engine windows.Handle) error {
	r, _, _ := procFwpmTransactionAbort.Call(uintptr(engine))
	return fwpmResultError("FwpmTransactionAbort", r)
}

// utf16LEBytes encodes s as little-endian UTF-16 bytes without a terminator,
// the form WFP expects for an ALE_APP_ID byte blob.
func utf16LEBytes(s string) []byte {
	u := utf16.Encode([]rune(s))
	out := make([]byte, len(u)*2)
	for i, v := range u {
		out[i*2] = byte(v)
		out[i*2+1] = byte(v >> 8)
	}
	return out
}

// sha256Sum318 hashes seed and returns the first 16 digest bytes.
func sha256Sum318(seed string) []byte {
	sum := sha256.Sum256([]byte(seed))
	return sum[:16]
}

// binaryLE32 reads a little-endian uint32 from b.
func binaryLE32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// addPermitLoopback installs a hard permit for this container's connections
// to the loopback proxy port.
func addPermitLoopback(engine windows.Handle, subLayer windows.GUID, layer windows.GUID, port uint16, pkg *windows.SID, ids *[]uint64) error {
	var pkgVal, isLoopback, portVal fwpConditionValue
	pkgVal.setSID(pkg)
	isLoopback.setUint32(fwpConditionFlagIsLoopback)
	portVal.setUint16(port)
	conds := []fwpmFilterCondition{
		{fieldKey: guidConditionALEPackageID, matchType: fwpmMatchEqual, conditionValue: pkgVal},
		{fieldKey: guidConditionFlags, matchType: fwpmMatchFlagsAllSet, conditionValue: isLoopback},
		{fieldKey: guidConditionIPRemotePort, matchType: fwpmMatchEqual, conditionValue: portVal},
	}
	f := fwpmFilter{
		displayData:         fwpmDisplayData{name: utf16Ptr("warden egress allow")},
		flags:               fwpmFilterFlagClearActionRight,
		layerKey:            layer,
		subLayerKey:         subLayer,
		weight:              fwpValue{valueType: fwpUint8, value: 15},
		numFilterConditions: uint32(len(conds)),
		filterCondition:     &conds[0],
		action:              fwpmAction{actionType: fwpmActionPermit},
	}
	return fwpmAdd(engine, &f, ids, "allow-loopback")
}

// addBlockAll blocks every other outbound connection from this container.
// The package-SID condition is essential: without it the filter would block
// outbound traffic for the whole host.
func addBlockAll(engine windows.Handle, subLayer windows.GUID, layer windows.GUID, pkg *windows.SID, ids *[]uint64) error {
	var pkgVal fwpConditionValue
	pkgVal.setSID(pkg)
	conds := []fwpmFilterCondition{{fieldKey: guidConditionALEPackageID, matchType: fwpmMatchEqual, conditionValue: pkgVal}}
	f := fwpmFilter{
		displayData:         fwpmDisplayData{name: utf16Ptr("warden egress deny-all")},
		layerKey:            layer,
		subLayerKey:         subLayer,
		weight:              fwpValue{valueType: fwpUint8, value: 0},
		numFilterConditions: uint32(len(conds)),
		filterCondition:     &conds[0],
		action:              fwpmAction{actionType: fwpmActionBlock},
	}
	return fwpmAdd(engine, &f, ids, "deny-all")
}

func fwpmAdd(engine windows.Handle, f *fwpmFilter, ids *[]uint64, what string) error {
	var id uint64
	r, _, _ := procFwpmFilterAdd.Call(
		uintptr(engine),
		uintptr(unsafe.Pointer(f)),
		0, // security descriptor
		uintptr(unsafe.Pointer(&id)),
	)
	if err := fwpmResultError("FwpmFilterAdd("+what+")", r); err != nil {
		return failClose("install WFP "+what+" filter", err)
	}
	*ids = append(*ids, id)
	return nil
}

// close removes installed filters and closes the engine. It always runs, even
// partway through a failed install, so nothing leaks into the next run.
func (s *wfpSession) close() {
	if s == nil || s.engine == 0 {
		return
	}
	for _, id := range s.filterIDs {
		_, _, _ = procFwpmFilterDeleteById.Call(uintptr(s.engine), uintptr(id))
	}
	_, _, _ = procFwpmEngineClose.Call(uintptr(s.engine))
	s.engine = 0
	s.filterIDs = nil
}
