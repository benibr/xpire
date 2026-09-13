// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import "testing"

func TestParseZfsGetOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "mountpoint", output: "xpire-pool/data\tmountpoint\t/mnt/zfs/data\tdefault\n", want: "/mnt/zfs/data"},
		{name: "mounted", output: "xpire-pool/data\tmounted\tyes\t-\n", want: "yes"},
		{name: "value-with-spaces", output: "xpire-pool/data\tcomment\tsome value\tlocal\n", want: "some value"},
		{name: "empty", output: "", want: ""},
		{name: "too-few-fields", output: "xpire-pool/data\tmounted\n", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseZfsGetOutput(tt.output); got != tt.want {
				t.Errorf("parseZfsGetOutput(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}

func TestMountpointUnder(t *testing.T) {
	tests := []struct {
		mountpoint string
		absPath    string
		want       bool
	}{
		{mountpoint: "/mnt/data", absPath: "/mnt/data", want: true},
		{mountpoint: "/mnt/data/sub", absPath: "/mnt/data", want: true},
		{mountpoint: "/mnt/data/sub", absPath: "/mnt/data/", want: true},
		{mountpoint: "/mnt/data", absPath: "/", want: true},
		{mountpoint: "/mnt/data01", absPath: "/mnt/data0", want: false},
		{mountpoint: "/mnt", absPath: "/mnt/data", want: false},
		{mountpoint: "none", absPath: "/mnt", want: false},
		{mountpoint: "legacy", absPath: "/", want: false},
	}
	for _, tt := range tests {
		if got := mountpointUnder(tt.mountpoint, tt.absPath); got != tt.want {
			t.Errorf("mountpointUnder(%q, %q) = %v, want %v", tt.mountpoint, tt.absPath, got, tt.want)
		}
	}
}
