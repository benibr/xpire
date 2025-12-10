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

func TestMain(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		// BTRFS
		// prune, non expired
		{[]string{"--path", "./tests/mnt/btrfs/subvolume01", "--prune"},
			"level=info msg=Detected filesystem: btrfs\nlevel=info msg=pruning expired data in './tests/mnt/btrfs/subvolume01'\n"},
		// set expire date
		{[]string{"--path", "./tests/mnt/btrfs/subvolume02", "--set", "2002-01-01 15:00:00"},
			"level=info msg=Detected filesystem: btrfs\nlevel=info msg=setting expiration date on './tests/mnt/btrfs/subvolume02' to 2002-01-01 15:00:00\n"},
		// prune, one expired
		{[]string{"--path", "./tests/mnt/btrfs/subvolume02", "--prune"},
			"level=info msg=Detected filesystem: btrfs\nlevel=info msg=pruning expired data in './tests/mnt/btrfs/subvolume02'\nlevel=info msg=↳ Subvolume 'subvolume02' expired since 2002-01-01 15:00:00\n"},
		// prune, one sub subvolume expired
		{[]string{"--path", "./tests/mnt/btrfs/subvolume03", "--prune"},
			"level=info msg=Detected filesystem: btrfs\nlevel=info msg=pruning expired data in './tests/mnt/btrfs/subvolume03'\nlevel=info msg=↳ Subvolume 'subvolume03/subvolume30' expired since 2002-01-01 15:00:00\n"},
		// prune on a non subvolume directory
		{[]string{"--path", "./tests/mnt/btrfs/dir", "--prune"},
			"level=info msg=Detected filesystem: btrfs\nlevel=info msg=pruning expired data in './tests/mnt/btrfs/dir'\n"},
		// prune, non-expired, on a subvolume mounted under different name
		{[]string{"--path", "./tests/mnt/btrfs/subvolume-mount", "--prune"},
			"level=info msg=Detected filesystem: btrfs\nlevel=info msg=pruning expired data in './tests/mnt/btrfs/subvolume-mount'\n"},
		// prune on subvolume with wrong time format in xattr
		{[]string{"--path", "./tests/mnt/btrfs/wrong-time-format", "--prune"},
			"level=info msg=Detected filesystem: btrfs\nlevel=info msg=pruning expired data in './tests/mnt/btrfs/wrong-time-format'\nlevel=warning msg=cannot parse expire date format:\n\tparsing time \"205-02 111\" as \"2006-01-02 15:04:05\": cannot parse \"205-02 111\" as \"2006\"\n"},
		// FIXME: add test for missing root permissions with btrfs prune
		// FIXME: add test for btrfs --set on subvolume where user access is forbidden

		// ZFS
		// prune, non expired
		{[]string{"--path", "./tests/mnt/zfs/dataset00/dataset01", "--prune"},
			"level=info msg=Detected filesystem: zfs\nlevel=info msg=pruning expired data in './tests/mnt/zfs/dataset00/dataset01'\n"},
		// set expire date
		{[]string{"--path", "./tests/mnt/zfs/dataset00/dataset01", "--set", "2002-01-01 15:00:00"},
			"level=info msg=Detected filesystem: zfs\nlevel=info msg=setting expiration date on './tests/mnt/zfs/dataset00/dataset01' to 2002-01-01 15:00:00\n"},
		// prune, one expired
		{[]string{"--path", "./tests/mnt/zfs/dataset00/dataset02", "--prune"},
			"level=info msg=Detected filesystem: zfs\nlevel=info msg=pruning expired data in './tests/mnt/zfs/dataset00/dataset02'\nlevel=info msg=↳ Dataset 'xpool/dataset00/dataset02' expired since 2002-01-01 15:00:00\n"},
		// prune, one sub dataset expired
		{[]string{"--path", "./tests/mnt/zfs/dataset00/dataset03", "--prune"},
			"level=info msg=Detected filesystem: zfs\nlevel=info msg=pruning expired data in './tests/mnt/zfs/dataset00/dataset03'\nlevel=info msg=↳ Dataset 'xpool/dataset00/dataset03/dataset33' expired since 2002-01-01 15:00:00\n"},
		// prune on a non dataset directory
		{[]string{"--path", "./tests/mnt/zfs/dir", "--prune"},
			"level=info msg=Detected filesystem: zfs\nlevel=info msg=pruning expired data in './tests/mnt/zfs/dir'\n"},
		// prune on subvolume with wrong time format in xattr
		{[]string{"--path", "./tests/mnt/zfs/dataset00/wrong-time-format", "--prune"},
			"level=info msg=Detected filesystem: zfs\nlevel=info msg=pruning expired data in './tests/mnt/zfs/dataset00/wrong-time-format'\nlevel=warning msg=Cannot parse expire date format:\n\tparsing time \"205-02 111\" as \"2006-01-02 15:04:05\": cannot parse \"205-02 111\" as \"2006\"\n"},
	}

	for _, tt := range tests {
		t.Run("Testing with args "+tt.args[0], func(t *testing.T) {
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
