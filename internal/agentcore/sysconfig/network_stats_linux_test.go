//go:build linux

package sysconfig

import "testing"

func TestValidLinuxInterfaceName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "eth0", want: true},
		{name: "tailscale0", want: true},
		{name: "bridge-name", want: true},
		{name: "", want: false},
		{name: ".", want: false},
		{name: "..", want: false},
		{name: "../etc", want: false},
		{name: `path\escape`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validLinuxInterfaceName(tt.name); got != tt.want {
				t.Fatalf("validLinuxInterfaceName(%q)=%v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
