//go:build integration

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
	"strings"
	"syscall"
	"testing"
)

// TestCLI checks argument handling and exit codes of xpire without the
// need of a supported filesystem or root permissions.
func TestCLI(t *testing.T) {
	dir := t.TempDir()
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		t.Fatal(err)
	}
	if stat.Type == 0x9123683E || stat.Type == 0x2FC12FC1 {
		t.Skipf("temporary directory '%s' is on a supported filesystem", dir)
	}

	tests := []struct {
		name   string
		args   []string
		rc     int
		output string
	}{
		{
			name:   "help",
			args:   []string{"--help"},
			rc:     RC_OK,
			output: "Usage: xpire",
		},
		{
			name:   "unknown-argument",
			args:   []string{"--unknown"},
			rc:     255,
			output: "unknown argument --unknown",
		},
		{
			name:   "missing-path",
			args:   []string{"--list"},
			rc:     RC_ERR_ARGS,
			output: "--path missing",
		},
		{
			name:   "invalid-loglevel",
			args:   []string{"--path", dir, "--loglevel", "invalid", "--list"},
			rc:     RC_ERR_ARGS,
			output: "not a valid logrus Level",
		},
		{
			name:   "unsupported-filesystem",
			args:   []string{"--path", dir, "--list"},
			rc:     RC_ERR_FS,
			output: "Filesystem not supported",
		},
		{
			name:   "non-existing-path",
			args:   []string{"--path", dir + "/missing", "--list"},
			rc:     RC_ERR_FS,
			output: "no such file or directory",
		},
		{
			name:   "unknown-plugin",
			args:   []string{"--path", dir, "--plugin", "unknown", "--list"},
			rc:     RC_ERR_PLUGIN,
			output: "filesystems/unknown/unknown.so",
		},
		{
			name:   "missing-action",
			args:   []string{"--path", dir, "--plugin", "btrfs"},
			rc:     RC_ERR_ARGS,
			output: "--set or --prune",
		},
		{
			name:   "dry-run-without-prune",
			args:   []string{"--path", dir, "--plugin", "btrfs", "--dry-run"},
			rc:     RC_ERR_ARGS,
			output: "--set or --prune",
		},
		{
			name:   "invalid-date",
			args:   []string{"--path", dir, "--plugin", "btrfs", "--set", "2002-01-01"},
			rc:     RC_ERR_ARGS,
			output: `parsing time "2002-01-01"`,
		},
		{
			name:   "set-with-wrong-plugin",
			args:   []string{"--path", dir, "--plugin", "btrfs", "--set", "2002-01-01 15:00:00"},
			rc:     RC_ERR_FS,
			output: "is not a btrfs subvolume",
		},
		{
			name:   "unset-with-wrong-plugin",
			args:   []string{"--path", dir, "-P", "btrfs", "--unset"},
			rc:     RC_ERR_FS,
			output: "is not a btrfs subvolume",
		},
		{
			name: "prune-with-wrong-plugin",
			args: []string{"--path", dir, "-P", "btrfs", "--prune"},
			rc:   RC_ERR_PLUGIN,
		},
		{
			name: "list-with-wrong-plugin",
			args: []string{"--path", dir, "-P", "btrfs", "--list"},
			rc:   RC_ERR_PLUGIN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, rc := runXpire(t, tt.args...)
			if rc != tt.rc {
				t.Errorf("want exit code %d, got %d\n%s", tt.rc, rc, out)
			}
			if !strings.Contains(out, tt.output) {
				t.Errorf("output does not contain %q\n%s", tt.output, out)
			}
		})
	}
}
