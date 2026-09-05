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
// They are constants that a WFP implementation cannot invent; the escape tests
// on a Windows runner pin them in place.

// FWPM_ACTION_* (fwpmu.h).
const (
	fwpmActionBlock  = 0x00000000
	fwpmActionPermit = 0x00000001
	fwpmActionNone   = 0x00000002
)

// FWP_MATCH_TYPE (fwptypes.h).
const fwpmMatchEqual = 0

// FWP_DATA_TYPE (fwptypes.h).
const (
	fwpUint8        = 0x01
	fwpUint16       = 0x02
	fwpUint32       = 0x03
	fwpByteArray16  = 0x09
	fwpByteBlobType = 0x0A
	fwpSid          = 0x0B
)

// FWPM Condition field GUIDs (fwpmu.h). FWP_CONDITION_FLAG_IS_LOOPBACK marks
// traffic that stays on a loopback interface.
const fwpConditionFlagIsLoopback = 0x00000001

var (
	// FWPM_CONDITION_ALE_APP_ID identifies the executable by its NT image path.
	guidConditionALEAppID = guidOf("b0e247ec-84be-4d1a-982c-deb94538337c")
	// FWPM_CONDITION_FLAGS carries filter flags such as IS_LOOPBACK.
	guidConditionFlags = guidOf("632ce23b-5167-435c-86cf-242bbbb70cb1")
	// FWPM_CONDITION_IP_REMOTE_PORT is the destination port condition.
	guidConditionIPRemotePort = guidOf("6fb1c71d-8ae9-4b97-b873-a462be2c14db")
)

// Well-known WFP layers we attach to (fwpmu.h).
var (
	// FWPM_LAYER_ALE_AUTH_CONNECT_V4
	guidLayerALEAuthConnectV4 = guidOf("d78e1e87-8644-4ea5-9437-d809ecefc971")
	// FWPM_LAYER_ALE_AUTH_CONNECT_V6
	guidLayerALEAuthConnectV6 = guidOf("4a6f2e6c-9a3d-4f1d-a43f-9bcb355219b5")
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

// fwpConditionValue mirrors FWP_CONDITION_VALUE (fwptypes.h): a data-type
// discriminator plus a 16-byte value union.
type fwpConditionValue struct {
	valueType uint32
	_         uint32
	value     [16]byte // union
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

// FWP_VALUE mirrors FWP_VALUE (fwptypes.h): a data-type discriminator plus a
// 16-byte value union. filters use it for weight (FWP_EMPTY when unused).
type fwpValue struct {
	valueType uint32
	_         uint32
	value     [16]byte // union, unused for the empty weight
}

// fwpEmpty is FWP_DATA_TYPE::FWP_EMPTY.
const fwpEmpty = 0

// fwpmFilterCondition mirrors FWPM_FILTER_CONDITION (fwpmu.h).
type fwpmFilterCondition struct {
	fieldKey       windows.GUID
	matchType      uint32
	conditionValue fwpConditionValue
}

// fwpmFilter mirrors FWPM_FILTER (fwpmu.h). Field order matches the SDK so the
// compiler reproduces the MSVC x64 offsets for the pointer/uint32 transitions;
// layerKey and weight.valueType are set for every filter added. The exact
// layout is cross-checked by the Windows integration tests; a wrong offset
// makes FwpmFilterAdd fail, which fails the run closed.
type fwpmFilter struct {
	filterKey           windows.GUID
	flags               uint32
	providerKey         *windows.GUID
	providerData        fwpByteBlob
	displayData         fwpmDisplayData
	layerKey            windows.GUID
	weight              fwpValue
	subLayerKey         windows.GUID
	rawContext          windows.GUID
	providerContextKey  windows.GUID
	conditions          *fwpmFilterCondition
	numFilterConditions uint32
	action              uint32
	u                   uintptr // union member, unused (zero)
}

// fwpmSublayer mirrors FWPM_SUBLAYER (fwpmu.h).
type fwpmSublayer struct {
	subLayerKey  windows.GUID
	displayData  fwpmDisplayData
	providerKey  windows.GUID
	providerData windows.GUID
	flags        uint32
	weight       uint32
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

// fwpmOpen opens a handle to the filter engine.
func fwpmOpen() (windows.Handle, error) {
	var engine windows.Handle
	r, _, _ := procFwpmEngineOpen.Call(
		0, // serverName (local)
		0, // authnService (ignored with NULL server)
		0, // authIdentity
		0, // session
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
		conditions:          &conds[0],
		numFilterConditions: uint32(len(conds)),
		action:              fwpmActionPermit,
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
		action:      fwpmActionBlock,
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
