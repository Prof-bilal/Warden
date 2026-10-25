//go:build windows

package windows

import (
	"reflect"
	"testing"
)

func TestAncestorDirs(t *testing.T) {
	got := ancestorDirs([]pathGrant{
		{Path: `C:\work\srv`},
		{Path: `C:\work\srv\out`},
		{Path: `D:\data`},
	})
	want := []string{`C:\work`, `C:\`, `D:\`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ancestorDirs = %q, want %q", got, want)
	}
}
