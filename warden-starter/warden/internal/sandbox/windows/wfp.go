//go:build windows

package windows

import (
	"crypto/sha256"
	"fmt"
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
// They are constants a WFP implementation cannot invent; wfp_spec_test.go and
// wfp_layout_test.go pin them against the published SDK values.

// The WFP action types, data types, condition flags and identifier strings
// live in wfp_spec.go, which has no build tag so tests on every platform can
// pin their values. The struct mirrors stay here because they embed
// windows.GUID.

// The built-in WFP identifiers the egress filters attach to.
var (
	guidConditionALEAppID     = guidOf(guidStringConditionALEAppID)
	guidConditionFlags        = guidOf(guidStringConditionFlags)
	guidConditionIPRemotePort = guidOf(guidStringConditionIPRemotePort)
	guidWardenActionType      = guidOf(guidStringWardenActionType)

	guidLayerALEAuthConnectV4 = guidOf(guidStringLayerALEAuthConnectV4)
	guidLayerALEAuthConnectV6 = guidOf(guidStringLayerALEAuthConnectV6)
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

// fwpConditionValue mirrors FWP_CONDITION_VALUE0 (fwptypes.h): a data-type
// discriminator, four bytes of padding, then the tagged union. The union holds
// an FWP_BYTE_ARRAY16 (16 bytes) and members needing 8-byte alignment, so it
// must be 8-byte aligned here too. A [16]byte union aligns to 1, which shrinks
// the enclosing FWPM_FILTER_CONDITION0 from 48 to 44 bytes and shifts every
// later filter field onto the wrong offset.
type fwpConditionValue struct {
	valueType uint32
	_         uint32
	value     [2]uint64 // union
}

func (v *fwpConditionValue) setUint32(x uint32) {
	v.valueType = fwpUint32
	*(*uint32)(unsafe.Pointer(&v.value[0])) = x
}

func (v *fwpConditionValue) setUint16(x uint16) {
	v.valueType = fwpUint16
	*(*uint16)(unsafe.Pointer(&v.value[0])) = x
}

func (v *fwpConditionValue) setByteBlob(b *fwpByteBlob) {
	v.valueType = fwpByteBlobType
	*(*uintptr)(unsafe.Pointer(&v.value[0])) = uintptr(unsafe.Pointer(b))
}

// fwpValue mirrors FWP_VALUE0 (fwptypes.h): a data-type discriminator, four
// bytes of padding, then an 8-byte union (its largest member is a pointer).
// FWP_VALUE0 is 16 bytes, not 24only FWP_CONDITION_VALUE0 carries the 16-byte
// FWP_BYTE_ARRAY16 member.
type fwpValue struct {
	valueType uint32
	_         uint32
	value     uint64 // union; unused for the empty weight
}

// fwpEmpty is FWP_DATA_TYPE::FWP_EMPTY.
const fwpEmpty = 0

// fwpmAction mirrors FWPM_ACTION0 (fwpmtypes.h): the action type plus a 16-byte
// GUID union. Non-callout actions (block/permit) require filterType; callout
// actions put the callout key in the same bytes. 24 bytes.
type fwpmAction struct {
	actionType uint32
	_          uint32
	filterType windows.GUID
}

// fwpmFilterCondition mirrors FWPM_FILTER_CONDITION0 (fwpmtypes.h). 48 bytes:
// the field GUID, the match type, four bytes of padding, then the 8-byte
// aligned condition value.
type fwpmFilterCondition struct {
	fieldKey       windows.GUID
	matchType      uint32
	_              uint32
	conditionValue fwpConditionValue
}

// fwpmFilter mirrors FWPM_FILTER0 (fwpmtypes.h). The field order and explicit
// padding reproduce the SDK's x64 layout (200 bytes); fwpm_layout_test.go
// asserts every offset. A mismatch makes FwpmFilterAdd0 fail with
// RPC_X_BAD_STUB_DATA (0x6f7), because BFE marshals the struct with its own
// definition and reads garbage wherever the two layouts diverge.
type fwpmFilter struct {
	filterKey           windows.GUID
	displayData         fwpmDisplayData
	flags               uint32
	_                   uint32
	providerKey         *windows.GUID
	providerData        fwpByteBlob
	layerKey            windows.GUID
	subLayerKey         windows.GUID
	weight              fwpValue
	numFilterConditions uint32
	_                   uint32
	filterCondition     *fwpmFilterCondition
	action              fwpmAction
	providerContext     [2]uint64 // union { UINT64 rawContext; GUID providerContextKey }
	reserved            *windows.GUID
	filterId            uint64
	effectiveWeight     fwpValue
}

// fwpmSublayer mirrors FWPM_SUBLAYER0 (fwpmtypes.h). 72 bytes: the sublayer
// GUID, display data, flags, a provider key pointer, a byte blob and a 16-bit
// weight, padded to the struct's 8-byte alignment.
type fwpmSublayer struct {
	subLayerKey  windows.GUID
	displayData  fwpmDisplayData
	flags        uint32
	_            uint32
	providerKey  *windows.GUID
	providerData fwpByteBlob
	weight       uint16
	_            [6]byte
}

// guidOf parses a registry-style GUID string into a windows.GUID. These are
// compile-time constants; a failure indicates an SDL regression, returned as
// nil GUID and caught by the layer checks in installWFPEgress.
func guidOf(s string) windows.GUID {
	g, err := windows.GUIDFromString(s)
	if err != nil {
		return windows.GUID{}
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
	// (0x6f7) on FwpmFilterAdd. That 0x6f7 was later reproduced on a real
	// Windows Server 2025 host and traced to a wrong FWPM_FILTER0 layout in
	// this file, not to the host; the structs and constants are now pinned by
	// wfp_layout_test.go and wfp_spec_test.go. The probe stays because
	// installing and rolling back a real filter is the only thing that proves
	// the host can enforce egress at all. Run the install path's exact sequence
	// against a throwaway sublayer and abort the transaction so nothing
	// persists: Supported() then reports the host's true ability to enforce
	// egress rules, and callers skip instead of failing on a refuse-to-run
	// sandbox.
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
	if err := addBlockAll(engine, subLayer, guidLayerALEAuthConnectV4, &probeIDs); err != nil {
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

// fwpmOpen opens a handle to the filter engine.
func fwpmOpen() (windows.Handle, error) {
	var engine windows.Handle
	r, _, _ := procFwpmEngineOpen.Call(
		0,             // serverName: NULL, local engine
		rpcAuthnWinNT, // authnService: RPC_C_AUTHN_WINNT (allowed values are WINNT and DEFAULT)
		0,             // authIdentity: NULL, use the calling thread's credentials
		0,             // session: NULL, default session flags
		uintptr(unsafe.Pointer(&engine)),
	)
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
	engine     windows.Handle
	subLayer   windows.GUID
	filterIDs  []uint64
	appBlobBuf []byte
	appBlob    fwpByteBlob
}

// installWFPEgress installs the WFP filters for one run. It fails closed: any
// error aborts the transaction and closes the engine so a partially configured
// policy can never leave a default-permit path.
func installWFPEgress(imageNTPath string, allowPort uint16, seedGUID string) (sess *wfpSession, err error) {
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

	sess.appBlobBuf = utf16LEBytes(imageNTPath)
	if len(sess.appBlobBuf) == 0 {
		return nil, failClose("WFP image path", fmt.Errorf("empty image path"))
	}
	sess.appBlob = fwpByteBlob{size: uint32(len(sess.appBlobBuf)), data: &sess.appBlobBuf[0]}

	if r, _, _ := procFwpmTransactionBegin.Call(uintptr(engine), 0); r != 0 {
		return nil, fwpmResultError("FwpmTransactionBegin", r)
	}

	sub := fwpmSublayer{subLayerKey: sess.subLayer, displayData: fwpmDisplayData{name: utf16Ptr("warden sublayer")}}
	if r, _, _ := procFwpmSublayerAdd.Call(uintptr(engine), uintptr(unsafe.Pointer(&sub)), 0); r != 0 {
		_ = fwpmAbort(engine)
		return nil, fwpmResultError("FwpmSublayerAdd", r)
	}

	for _, layer := range [2]windows.GUID{guidLayerALEAuthConnectV4, guidLayerALEAuthConnectV6} {
		// Hard-permit: the loopback proxy bridge from this image.
		if err := addPermitLoopback(engine, sess.subLayer, layer, allowPort, &sess.appBlob, &sess.filterIDs); err != nil {
			_ = fwpmAbort(engine)
			return nil, err
		}
		// Hard-block every other outbound connection for the image. UDP/DNS is
		// denied by the AppContainer token; this closes the TCP path.
		if err := addBlockAll(engine, sess.subLayer, layer, &sess.filterIDs); err != nil {
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

// addPermitLoopback installs a permit filter allowing loopback connections from
// the sandbox image to the proxy port.
func addPermitLoopback(engine windows.Handle, subLayer windows.GUID, layer windows.GUID, port uint16, app *fwpByteBlob, ids *[]uint64) error {
	conds := make([]fwpmFilterCondition, 0, 3)
	var isLoopback fwpConditionValue
	isLoopback.setUint32(fwpConditionFlagIsLoopback)
	conds = append(conds, fwpmFilterCondition{fieldKey: guidConditionFlags, matchType: fwpmMatchEqual, conditionValue: isLoopback})

	var portVal fwpConditionValue
	portVal.setUint16(port)
	conds = append(conds, fwpmFilterCondition{fieldKey: guidConditionIPRemotePort, matchType: fwpmMatchEqual, conditionValue: portVal})

	var appVal fwpConditionValue
	appVal.setByteBlob(app)
	conds = append(conds, fwpmFilterCondition{fieldKey: guidConditionALEAppID, matchType: fwpmMatchEqual, conditionValue: appVal})

	f := fwpmFilter{
		displayData:         fwpmDisplayData{name: utf16Ptr("warden egress allow")},
		layerKey:            layer,
		weight:              fwpValue{valueType: fwpEmpty},
		subLayerKey:         subLayer,
		filterCondition:     &conds[0],
		numFilterConditions: uint32(len(conds)),
		action:              fwpmAction{actionType: fwpmActionPermit, filterType: guidWardenActionType},
	}
	return fwpmAdd(engine, &f, ids, "allow-loopback")
}

// addBlockAll installs a deny-all outbound filter on a layer.
func addBlockAll(engine windows.Handle, subLayer windows.GUID, layer windows.GUID, ids *[]uint64) error {
	f := fwpmFilter{
		displayData: fwpmDisplayData{name: utf16Ptr("warden egress deny-all")},
		layerKey:    layer,
		weight:      fwpValue{valueType: fwpEmpty},
		subLayerKey: subLayer,
		action:      fwpmAction{actionType: fwpmActionBlock, filterType: guidWardenActionType},
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
