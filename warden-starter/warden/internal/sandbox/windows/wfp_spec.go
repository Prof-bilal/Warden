package windows

// Windows Filtering Platform constants copied from fwptypes.h and fwpmu.h.
//
// This file deliberately carries no build tag. Every value here is a plain
// integer or GUID string, so tests on any platform can assert it. That matters:
// a wrong constant is invisible until FwpmFilterAdd0 rejects the filter at run
// time, and a wrong FWPM_CONDITION_* GUID matches nothing at all, so the filter
// installs and silently enforces nothing.
//
// These values were previously guessed and every one of them was wrong. They
// are now pinned by wfp_spec_test.go against the published SDK values.

// FWP_ACTION_TYPE (fwptypes.h). Block and permit are terminating actions, so
// they carry FWP_ACTION_FLAG_TERMINATING (0x1000). The bare 0x1/0x2 values are
// not valid action types and are rejected by WFP.
const (
	fwpActionFlagTerminating = 0x00001000
	fwpmActionBlock          = 0x0001 | fwpActionFlagTerminating // 0x1001
	fwpmActionPermit         = 0x0002 | fwpActionFlagTerminating // 0x1002
)

// FWP_MATCH_TYPE (fwptypes.h).
const fwpmMatchEqual = 0

// FWP_DATA_TYPE (fwptypes.h). The enumerators are positional:
// FWP_EMPTY=0, UINT8, UINT16, UINT32, UINT64, INT8, INT16, INT32, INT64,
// FLOAT, DOUBLE, BYTE_ARRAY16_TYPE=11, BYTE_BLOB_TYPE=12, SID=13.
const (
	fwpUint8        = 0x01
	fwpUint16       = 0x02
	fwpUint32       = 0x03
	fwpByteArray16  = 0x0B
	fwpByteBlobType = 0x0C
	fwpSid          = 0x0D
)

// FWP_CONDITION_FLAG_IS_LOOPBACK (fwptypes.h): the traffic stays on a loopback
// interface.
const fwpConditionFlagIsLoopback = 0x00000001

// Built-in WFP identifier GUIDs (fwpmu.h).
const (
	// FWPM_LAYER_ALE_AUTH_CONNECT_V4 / V6: the ALE connect layers that carry
	// outbound IPv4/IPv6 connect authorization.
	guidStringLayerALEAuthConnectV4 = "c38d57d1-05a7-4c33-904f-7fbceee60e82"
	guidStringLayerALEAuthConnectV6 = "4a72393b-319f-44bc-84c3-ba54dcb3b6b4"

	// FWPM_CONDITION_ALE_APP_ID identifies the executable by its NT image path.
	guidStringConditionALEAppID = "d78e1e87-8644-4ea5-9437-d809ecefc971"
	// FWPM_CONDITION_FLAGS carries filter flags such as IS_LOOPBACK.
	guidStringConditionFlags = "632ce23b-5167-435c-86d7-e903684aa80c"
	// FWPM_CONDITION_IP_REMOTE_PORT is the destination port condition.
	guidStringConditionIPRemotePort = "c35a604d-d22b-4e1a-91b4-68f674ee674b"
)

// guidStringWardenActionType is the provider-chosen action GUID that WFP
// requires for non-callout (block/permit) filters. The value is arbitrary but
// must be set.
const guidStringWardenActionType = "8f2c1d4e-5b6a-4c7d-9e8f-0a1b2c3d4e5f"
