package main

import (
		"os/exec"
		"testing"
)

func TestMain(t *testing.T) {
		tests := []struct {
				args		[]string
				want		string
		}{
				// prune, non expired
				{[]string{"--path", "./tests/mnt/btrfs", "--prune"},
				 "level=info msg=\"detected filesystem: btrfs\"\nlevel=info msg=\"pruning all expired snapshots in './tests/mnt/btrfs'\"\n"},
				// set expire date
				 {[]string{"--path", "./tests/mnt/btrfs/subvolume", "--set", "2002-01-01 15:00:00"},
				 "level=info msg=\"detected filesystem: btrfs\"\nlevel=info msg=\"setting expiration date on snapshot './tests/mnt/btrfs/subvolume' to 2002-01-01 15:00:00\"\n"},
				// prune, one expired
				//{[]string{"--path", "./tests/mnt/btrfs", "--prune"},
				// "level=info msg=\"detected filesystem: btrfs\"\npruning all expired snapshots in './tests/mnt/btrfs'\n"},
		}

		for _, tt := range tests {
				t.Run("Testing with args "+tt.args[0], func(t *testing.T) {
						cmd := exec.Command("go", "run", "main.go")
						cmd.Args = append(cmd.Args, tt.args...)

						output, err := cmd.CombinedOutput()
						if got := string(output); got != tt.want {
								t.Errorf("\n got output: '%v'\nwant output: '%v'", got, tt.want)
						}
						if err != nil {
								t.Fatalf("Failed to run command: %v", err)
						}

				})
		}
}
