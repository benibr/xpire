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

func TestSubvolumeUnder(t *testing.T) {
	tests := []struct {
		svPath  string
		relPath string
		want    bool
	}{
		{svPath: "data", relPath: "", want: true},
		{svPath: "data/sub", relPath: "", want: true},
		{svPath: "data", relPath: "data", want: true},
		{svPath: "data/sub", relPath: "data", want: true},
		{svPath: "data/sub/subsub", relPath: "data", want: true},
		{svPath: "data01", relPath: "data0", want: false},
		{svPath: "data0/sub", relPath: "data01", want: false},
		{svPath: "data", relPath: "data/sub", want: false},
		{svPath: "other", relPath: "data", want: false},
	}
	for _, tt := range tests {
		if got := subvolumeUnder(tt.svPath, tt.relPath); got != tt.want {
			t.Errorf("subvolumeUnder(%q, %q) = %v, want %v", tt.svPath, tt.relPath, got, tt.want)
		}
	}
}

func TestSubvolumeRelPath(t *testing.T) {
	tests := []struct {
		name       string
		absPath    string
		mountPoint string
		mountRoot  string
		want       string
	}{
		{name: "mountpoint", absPath: "/mnt", mountPoint: "/mnt", mountRoot: "", want: ""},
		{name: "below-mountpoint", absPath: "/mnt/data/sub", mountPoint: "/mnt", mountRoot: "", want: "data/sub"},
		{name: "subvol-mountpoint", absPath: "/mnt", mountPoint: "/mnt", mountRoot: "vol", want: "vol"},
		{name: "below-subvol-mountpoint", absPath: "/mnt/sub", mountPoint: "/mnt", mountRoot: "vol/nested", want: "vol/nested/sub"},
		{name: "root-mountpoint", absPath: "/data", mountPoint: "/", mountRoot: "", want: "data"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := subvolumeRelPath(tt.absPath, tt.mountPoint, tt.mountRoot); got != tt.want {
				t.Errorf("subvolumeRelPath(%q, %q, %q) = %q, want %q", tt.absPath, tt.mountPoint, tt.mountRoot, got, tt.want)
			}
		})
	}
}

func TestSubvolumeFullPath(t *testing.T) {
	tests := []struct {
		name       string
		mountPoint string
		mountRoot  string
		svPath     string
		want       string
	}{
		{name: "top-level-mount", mountPoint: "/mnt", mountRoot: "", svPath: "data/sub", want: "/mnt/data/sub"},
		{name: "subvol-mount-itself", mountPoint: "/mnt", mountRoot: "vol", svPath: "vol", want: "/mnt"},
		{name: "below-subvol-mount", mountPoint: "/mnt", mountRoot: "vol", svPath: "vol/sub", want: "/mnt/sub"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := subvolumeFullPath(tt.mountPoint, tt.mountRoot, tt.svPath); got != tt.want {
				t.Errorf("subvolumeFullPath(%q, %q, %q) = %q, want %q", tt.mountPoint, tt.mountRoot, tt.svPath, got, tt.want)
			}
		})
	}
}
