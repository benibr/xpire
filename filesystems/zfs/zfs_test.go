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

import (
	"path/filepath"
	"strings"
	"testing"
)

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
		{name: "no-value", output: "xpire-pool/data@snap\tmountpoint\t-\t-\n", want: "-"},
		{name: "value-with-outer-spaces", output: "xpire-pool/data\tmountpoint\t /mnt/zfs/data \tlocal\n", want: " /mnt/zfs/data "},
		{name: "only-first-line", output: "xpire-pool/a\tmounted\tyes\t-\nxpire-pool/b\tmounted\tno\t-\n", want: "yes"},
		// KNOWN ISSUE K12 (TEST_PLAN.md): a mountpoint with a tab is cut off,
		// the xattr is then read from another directory than the dataset
		{name: "value-with-tab", output: "xpire-pool/data\tmountpoint\t/mnt/zfs/da\tta\tlocal\n", want: "/mnt/zfs/da\tta"},
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
		{mountpoint: "-", absPath: "/", want: false},
		{mountpoint: "/mnt/data/", absPath: "/mnt/data", want: true},
		{mountpoint: "/mnt/datax", absPath: "/mnt/data", want: false},
		{mountpoint: "/mnt/Data", absPath: "/mnt/data", want: false},
		{mountpoint: "/mnt/data", absPath: "/mnt/data/sub/deep", want: false},
		{mountpoint: "/data", absPath: "/mnt/data", want: false},
		{mountpoint: "mnt/data", absPath: "/mnt", want: false},
		// Documents current behaviour: every absolute mountpoint lies under
		// an empty path, which is what makes K1 in TEST_PLAN.md dangerous
		{mountpoint: "/mnt/data", absPath: "", want: true},
		{mountpoint: "", absPath: "", want: true},
		{mountpoint: "", absPath: "/mnt", want: false},
	}
	for _, tt := range tests {
		if got := mountpointUnder(tt.mountpoint, tt.absPath); got != tt.want {
			t.Errorf("mountpointUnder(%q, %q) = %v, want %v", tt.mountpoint, tt.absPath, got, tt.want)
		}
	}
}

// FuzzMountpointUnder checks with clean absolute paths, as CleanPath and
// zfs return them, that a mountpoint "under" a path really is that path or
// lies below it.
// Run the fuzzer:  go test ./filesystems/zfs/ -fuzz FuzzMountpointUnder
func FuzzMountpointUnder(f *testing.F) {
	f.Add("/mnt/data", "/mnt/data")
	f.Add("/mnt/data/sub", "/mnt/data")
	f.Add("/mnt/data01", "/mnt/data0")
	f.Add("/mnt", "/mnt/data")
	f.Add("/mnt/data", "/")
	f.Fuzz(func(t *testing.T, mountpoint, absPath string) {
		if !filepath.IsAbs(mountpoint) || !filepath.IsAbs(absPath) ||
			filepath.Clean(mountpoint) != mountpoint || filepath.Clean(absPath) != absPath {
			t.Skip("not a clean absolute path")
		}
		rel, err := filepath.Rel(absPath, mountpoint)
		below := err == nil && rel != ".." && !strings.HasPrefix(rel, "../")
		if got := mountpointUnder(mountpoint, absPath); got != below {
			t.Errorf("mountpointUnder(%q, %q) = %v, want %v", mountpoint, absPath, got, below)
		}
	})
}
