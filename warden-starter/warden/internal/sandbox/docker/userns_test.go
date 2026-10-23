package docker

import (
	"strings"
	"testing"
)

func TestParseIDMapAndMapID(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		id     uint64
		want   uint64
		wantOK bool
	}{
		{
			name:   "identity map",
			input:  "0 0 4294967295\n",
			id:     1000,
			want:   1000,
			wantOK: true,
		},
		{
			name:   "root in user namespace maps to host user",
			input:  "0 1000 1\n",
			id:     0,
			want:   1000,
			wantOK: true,
		},
		{
			name:   "unmapped namespace id",
			input:  "0 1000 1\n",
			id:     1,
			wantOK: false,
		},
		{
			name:   "offset range",
			input:  "1000 5000 100\n",
			id:     1050,
			want:   5050,
			wantOK: true,
		},
		{
			name:   "range end is exclusive",
			input:  "1000 5000 100\n",
			id:     1100,
			wantOK: false,
		},
		{
			name:   "multiple ranges",
			input:  "0 1000 1\n1 2000 10\n",
			id:     5,
			want:   2004,
			wantOK: true,
		},
		{
			name:   "empty map",
			input:  "\n\n",
			id:     0,
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := parseIDMap(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("parseIDMap: %v", err)
			}
			got, ok := mapID(tt.id, entries)
			if ok != tt.wantOK || (ok && got != tt.want) {
				t.Fatalf("mapID(%d) = (%d, %v), want (%d, %v)", tt.id, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestParseIDMapRejectsMalformed(t *testing.T) {
	for _, input := range []string{
		"0 1000\n",
		"a b c\n",
		"0 1000 -1\n",
		"0 1000 1 extra\n",
	} {
		if _, err := parseIDMap(strings.NewReader(input)); err == nil {
			t.Fatalf("parseIDMap(%q) succeeded, want error", input)
		}
	}
}
