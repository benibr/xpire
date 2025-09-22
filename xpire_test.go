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
		// prune, non expired
		{[]string{"--path", "./tests/mnt/btrfs", "--prune"},
			"level=info msg=\"Detected filesystem: btrfs\"\nlevel=info msg=\"pruning expired data in './tests/mnt/btrfs'\"\n"},
		// set expire date
		{[]string{"--path", "./tests/mnt/btrfs/subvolume", "--set", "2002-01-01 15:00:00"},
			"level=info msg=\"Detected filesystem: btrfs\"\nlevel=info msg=\"setting expiration date on './tests/mnt/btrfs/subvolume' to 2002-01-01 15:00:00\"\n"},
		// prune, one expired
		{[]string{"--path", "./tests/mnt/btrfs", "--prune"},
			"level=info msg=\"Detected filesystem: btrfs\"\nlevel=info msg=\"pruning expired data in './tests/mnt/btrfs'\"\nlevel=info msg=\"↳ 'tests/mnt/btrfs/subvolume' expired since 2002-01-01 15:00:00\"\n"},
		// FIXME: add test for missing root permissions with btrfs prune
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
