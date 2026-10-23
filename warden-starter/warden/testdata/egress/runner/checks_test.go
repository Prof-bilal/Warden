package main

import "testing"

func TestHostNoPort(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"example.test", "example.test"},
		{"example.test:8080", "example.test"},
		{"Example.Test.:443", "example.test"},
		{"[2001:db8::1]:443", "2001:db8::1"},
		{"2001:db8::1", "2001:db8::1"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := hostNoPort(tt.in); got != tt.want {
			t.Errorf("hostNoPort(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
