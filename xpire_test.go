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
	"os/exec"
	"testing"
)

// TestBTRFS runs all BTRFS plugin tests.
// Run all BTRFS tests:      go test -run TestBTRFS
// Run a single test:        go test -run "TestBTRFS/list"
func TestBTRFS(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "list",
			args: []string{"--path", "./tests/mnt/btrfs/subvolume01", "--list"},
			want: "level=info msg=searching for all expire dates in '/home/bbraunger/Workspace/private/xpire/tests/mnt/btrfs/subvolume01'\n",
		},
		{
			name: "prune-non-expired",
			args: []string{"--path", "./tests/mnt/btrfs/subvolume01", "--prune"},
			want: "level=info msg=pruning expired data in './tests/mnt/btrfs/subvolume01'\n",
		},
		{
			name: "set-expire-date",
			args: []string{"--path", "./tests/mnt/btrfs/subvolume02", "--set", "2002-01-01 15:00:00"},
			want: "level=info msg=setting expiration date on './tests/mnt/btrfs/subvolume02' to 2002-01-01 15:00:00\n",
		},
		{
			name: "unset-non-existing-expire-date",
			args: []string{"--path", "./tests/mnt/btrfs/subvolume03/subvolume31", "--unset"},
			want: "level=info msg=unsetting expiration date on './tests/mnt/btrfs/subvolume03/subvolume31'\n",
		},
		{
			name: "prune-one-expired",
			args: []string{"--path", "./tests/mnt/btrfs/subvolume02", "--prune"},
			want: "level=info msg=pruning expired data in './tests/mnt/btrfs/subvolume02'\nlevel=info msg=↳ Subvolume 'subvolume02' expired since 2002-01-01 15:00:00\n",
		},
		{
			name: "prune-one-sub-subvolume-expired",
			args: []string{"--path", "./tests/mnt/btrfs/subvolume03", "--prune"},
			want: "level=info msg=pruning expired data in './tests/mnt/btrfs/subvolume03'\nlevel=info msg=↳ Subvolume 'subvolume03/subvolume30' expired since 2002-01-01 15:00:00\n",
		},
		{
			name: "prune-on-non-subvolume-directory",
			args: []string{"--path", "./tests/mnt/btrfs/dir", "--prune"},
			want: "level=info msg=pruning expired data in './tests/mnt/btrfs/dir'\n",
		},
		{
			name: "prune-non-expired-mounted-under-different-name",
			args: []string{"--path", "./tests/mnt/btrfs/subvolume-mount", "--prune"},
			want: "level=info msg=pruning expired data in './tests/mnt/btrfs/subvolume-mount'\n",
		},
		{
			name: "prune-wrong-time-format",
			args: []string{"--path", "./tests/mnt/btrfs/wrong-time-format", "--prune"},
			want: "level=info msg=pruning expired data in './tests/mnt/btrfs/wrong-time-format'\nlevel=warning msg=cannot parse expire date format:\n\tparsing time \"205-02 111\" as \"2006-01-02 15:04:05\": cannot parse \"205-02 111\" as \"2006\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", ".")
			cmd.Args = append(cmd.Args, tt.args...)

			output, err := cmd.CombinedOutput()
			if got := string(output); got != tt.want {
				t.Errorf("\n want output: '%v'\ngot output: '%v'", tt.want, got)
			}
			if err != nil {
				t.Fatalf("Failed to run command: %v", err)
			}
		})
	}
}

// TestZFS runs all ZFS plugin tests.
// Run all ZFS tests:       go test -run TestZFS
// Run a single test:        go test -run "TestZFS/list"
func TestZFS(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "list",
			args: []string{"--path", "./tests/mnt/zfs/dataset00/dataset01", "--list"},
			want: "level=info msg=Detected filesystem: zfs\nlevel=info msg=Listing data in './tests/mnt/zfs/dataset00/dataset01'\n",
		},
		{
			name: "prune-non-expired",
			args: []string{"--path", "./tests/mnt/zfs/dataset00/dataset01", "--prune"},
			want: "level=info msg=Detected filesystem: zfs\nlevel=info msg=pruning expired data in './tests/mnt/zfs/dataset00/dataset01'\n",
		},
		{
			name: "set-expire-date",
			args: []string{"--path", "./tests/mnt/zfs/dataset00/dataset01", "--set", "2002-01-01 15:00:00"},
			want: "level=info msg=Detected filesystem: zfs\nlevel=info msg=setting expiration date on './tests/mnt/zfs/dataset00/dataset01' to 2002-01-01 15:00:00\n",
		},
		{
			name: "unset-non-existing-expire-date",
			args: []string{"--path", "./tests/mnt/zfs/dataset00/dataset01", "--unset"},
			want: "level=info msg=Detected filesystem: zfs\nlevel=info msg=unsetting expiration date on './tests/mnt/zfs/dataset00/dataset01'\n",
		},
		{
			name: "prune-one-expired",
			args: []string{"--path", "./tests/mnt/zfs/dataset00/dataset02", "--prune"},
			want: "level=info msg=Detected filesystem: zfs\nlevel=info msg=pruning expired data in './tests/mnt/zfs/dataset00/dataset02'\nlevel=info msg=↳ Dataset 'xpool/dataset00/dataset02' expired since 2002-01-01 15:00:00\n",
		},
		{
			name: "prune-one-sub-dataset-expired",
			args: []string{"--path", "./tests/mnt/zfs/dataset00/dataset03", "--prune"},
			want: "level=info msg=Detected filesystem: zfs\nlevel=info msg=pruning expired data in './tests/mnt/zfs/dataset00/dataset03'\nlevel=info msg=↳ Dataset 'xpool/dataset00/dataset03/dataset33' expired since 2002-01-01 15:00:00\n",
		},
		{
			name: "prune-on-non-dataset-directory",
			args: []string{"--path", "./tests/mnt/zfs/dir", "--prune"},
			want: "level=info msg=Detected filesystem: zfs\nlevel=info msg=pruning expired data in './tests/mnt/zfs/dir'\n",
		},
		{
			name: "prune-wrong-time-format",
			args: []string{"--path", "./tests/mnt/zfs/dataset00/wrong-time-format", "--prune"},
			want: "level=info msg=Detected filesystem: zfs\nlevel=info msg=pruning expired data in './tests/mnt/zfs/dataset00/wrong-time-format'\nlevel=warning msg=cannot parse expire date format:\n\tparsing time \"205-02 111\" as \"2006-01-02 15:04:05\": cannot parse \"205-02 111\" as \"2006\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", ".")
			cmd.Args = append(cmd.Args, tt.args...)

			output, err := cmd.CombinedOutput()
			if got := string(output); got != tt.want {
				t.Errorf("\n want output: '%v'\ngot output: '%v'", tt.want, got)
			}
			if err != nil {
				t.Fatalf("Failed to run command: %v", err)
			}
		})
	}
}
