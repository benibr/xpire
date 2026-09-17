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
		{svPath: "data/subsub", relPath: "data/sub", want: false},
		{svPath: "Data", relPath: "data", want: false},
		{svPath: "", relPath: "data", want: false},
		{svPath: "data", relPath: "data/sub/deep", want: false},
		{svPath: "nested/data", relPath: "data", want: false},
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
		{name: "below-root-mountpoint", absPath: "/data/sub/deep", mountPoint: "/", mountRoot: "", want: "data/sub/deep"},
		{name: "root-mountpoint-with-mount-root", absPath: "/home", mountPoint: "/", mountRoot: "@", want: "@/home"},
		{name: "mountpoint-name-repeated", absPath: "/mnt/mnt/sub", mountPoint: "/mnt", mountRoot: "", want: "mnt/sub"},
		{name: "nested-mount-root", absPath: "/home/user", mountPoint: "/home", mountRoot: "@/home", want: "@/home/user"},
		// Documents current behaviour: an empty path means the whole mount,
		// which is what makes K1 in TEST_PLAN.md dangerous
		{name: "empty-path", absPath: "", mountPoint: "/mnt", mountRoot: "", want: ""},
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
		{name: "nested-mount-root", mountPoint: "/home", mountRoot: "@/home", svPath: "@/home/user", want: "/home/user"},
		{name: "root-mountpoint", mountPoint: "/", mountRoot: "@", svPath: "@/data", want: "/data"},
		{name: "mount-root-name-repeated", mountPoint: "/mnt", mountRoot: "vol", svPath: "vol/vol/sub", want: "/mnt/vol/sub"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := subvolumeFullPath(tt.mountPoint, tt.mountRoot, tt.svPath); got != tt.want {
				t.Errorf("subvolumeFullPath(%q, %q, %q) = %q, want %q", tt.mountPoint, tt.mountRoot, tt.svPath, got, tt.want)
			}
		})
	}
}

// KNOWN ISSUE K4 (TEST_PLAN.md): the mountpoint is removed anywhere in the
// path, not only in front. A path outside of the mountpoint, as
// helpers.FindParentMount returns it for '/mntfoo', silently becomes the
// name of a subvolume that has nothing to do with the path.
func TestSubvolumeRelPathOutsideMountpoint(t *testing.T) {
	tests := []struct {
		name       string
		absPath    string
		mountPoint string
		unwanted   string
	}{
		{name: "sibling-with-same-prefix", absPath: "/mntfoo/sub", mountPoint: "/mnt", unwanted: "foo/sub"},
		{name: "mountpoint-inside-path", absPath: "/data/mnt/sub", mountPoint: "/mnt", unwanted: "data/sub"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := subvolumeRelPath(tt.absPath, tt.mountPoint, ""); got == tt.unwanted {
				t.Errorf("subvolumeRelPath(%q, %q, \"\") = %q, a subvolume the path does not point to", tt.absPath, tt.mountPoint, got)
			}
		})
	}
}

// KNOWN ISSUE K13 (TEST_PLAN.md): the mount root is removed as a string, not
// as a path. A subvolume outside of the mount root is mapped to the path of
// another subvolume. Not reachable today because findChildSubvolumes only
// returns subvolumes below the mount root.
func TestSubvolumeFullPathOutsideMountRoot(t *testing.T) {
	// 'vol/ume01/sub' is what is really mounted at '/mnt/ume01/sub'
	if got := subvolumeFullPath("/mnt", "vol", "volume01/sub"); got == "/mnt/ume01/sub" {
		t.Errorf("subvolumeFullPath(\"/mnt\", \"vol\", \"volume01/sub\") = %q, the path of another subvolume", got)
	}
}

// subvolumeFullPath must be the inverse of subvolumeRelPath, otherwise the
// subvolume that is deleted is not the one that was selected
func TestSubvolumePathRoundTrip(t *testing.T) {
	tests := []struct {
		mountPoint string
		mountRoot  string
		absPath    string
	}{
		{mountPoint: "/mnt", mountRoot: "", absPath: "/mnt/data"},
		{mountPoint: "/mnt", mountRoot: "", absPath: "/mnt/data/sub/deep"},
		{mountPoint: "/mnt", mountRoot: "", absPath: "/mnt/mnt"},
		{mountPoint: "/mnt", mountRoot: "vol", absPath: "/mnt"},
		{mountPoint: "/mnt", mountRoot: "vol", absPath: "/mnt/vol"},
		{mountPoint: "/mnt", mountRoot: "vol", absPath: "/mnt/volume"},
		{mountPoint: "/mnt", mountRoot: "vol/nested", absPath: "/mnt/sub"},
		{mountPoint: "/mnt/btrfs/subvolume-mount", mountRoot: "subvolume01", absPath: "/mnt/btrfs/subvolume-mount/subvolume01"},
		{mountPoint: "/", mountRoot: "", absPath: "/data"},
		{mountPoint: "/", mountRoot: "@", absPath: "/data/sub"},
		{mountPoint: "/home", mountRoot: "@/home", absPath: "/home/user name/-rf"},
	}
	for _, tt := range tests {
		relPath := subvolumeRelPath(tt.absPath, tt.mountPoint, tt.mountRoot)
		if got := subvolumeFullPath(tt.mountPoint, tt.mountRoot, relPath); got != tt.absPath {
			t.Errorf("round trip of %q with mountpoint %q and mount root %q = %q via %q",
				tt.absPath, tt.mountPoint, tt.mountRoot, got, relPath)
		}
	}
}
