package windows

import "testing"

// These tests pin the WFP constants to the published SDK values. They run on
// every platform, including CI, because the values live in a file with no build
// tag. A wrong action type or identifier GUID is otherwise invisible: WFP either
// rejects the filter at run time or silently matches nothing.

func TestWFPSpecActionAndDataTypeValues(t *testing.T) {
	// FWP_ACTION_BLOCK / FWP_ACTION_PERMIT are terminating actions.
	if fwpmActionBlock != 0x1001 {
		t.Errorf("fwpmActionBlock = %#x, want 0x1001", fwpmActionBlock)
	}
	if fwpmActionPermit != 0x1002 {
		t.Errorf("fwpmActionPermit = %#x, want 0x1002", fwpmActionPermit)
	}
	if fwpActionFlagTerminating != 0x1000 {
		t.Errorf("fwpActionFlagTerminating = %#x, want 0x1000", fwpActionFlagTerminating)
	}

	// FWP_DATA_TYPE is a positional enumerator: BYTE_BLOB_TYPE is 12, not 10.
	if fwpUint8 != 1 || fwpUint16 != 2 || fwpUint32 != 3 {
		t.Errorf("integer data types = %d/%d/%d, want 1/2/3", fwpUint8, fwpUint16, fwpUint32)
	}
	if fwpByteArray16 != 11 {
		t.Errorf("fwpByteArray16 = %d, want 11 (FWP_BYTE_ARRAY16_TYPE)", fwpByteArray16)
	}
	if fwpByteBlobType != 12 {
		t.Errorf("fwpByteBlobType = %d, want 12 (FWP_BYTE_BLOB_TYPE)", fwpByteBlobType)
	}
	if fwpSid != 13 {
		t.Errorf("fwpSid = %d, want 13 (FWP_SID)", fwpSid)
	}

	if fwpmMatchEqual != 0 {
		t.Errorf("fwpmMatchEqual = %d, want 0", fwpmMatchEqual)
	}
	if fwpConditionFlagIsLoopback != 1 {
		t.Errorf("fwpConditionFlagIsLoopback = %d, want 1", fwpConditionFlagIsLoopback)
	}
}

func TestWFPSpecIdentifierGUIDs(t *testing.T) {
	// Values from fwpmu.h. A layer or condition GUID that does not match the
	// SDK makes the filter either fail to install or match nothing.
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"FWPM_LAYER_ALE_AUTH_CONNECT_V4", guidStringLayerALEAuthConnectV4, "c38d57d1-05a7-4c33-904f-7fbceee60e82"},
		{"FWPM_LAYER_ALE_AUTH_CONNECT_V6", guidStringLayerALEAuthConnectV6, "4a72393b-319f-44bc-84c3-ba54dcb3b6b4"},
		{"FWPM_CONDITION_ALE_APP_ID", guidStringConditionALEAppID, "d78e1e87-8644-4ea5-9437-d809ecefc971"},
		{"FWPM_CONDITION_FLAGS", guidStringConditionFlags, "632ce23b-5167-435c-86d7-e903684aa80c"},
		{"FWPM_CONDITION_IP_REMOTE_PORT", guidStringConditionIPRemotePort, "c35a604d-d22b-4e1a-91b4-68f674ee674b"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}
