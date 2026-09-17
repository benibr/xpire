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
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestGetFsType(t *testing.T) {
	t.Run("unsupported-filesystem", func(t *testing.T) {
		dir := t.TempDir()
		var stat syscall.Statfs_t
		if err := syscall.Statfs(dir, &stat); err != nil {
			t.Fatal(err)
		}
		if stat.Type == 0x9123683E || stat.Type == 0x2FC12FC1 {
			t.Skipf("temporary directory '%s' is on a supported filesystem", dir)
		}
		_, err := getFsType(dir)
		if err == nil || !strings.Contains(err.Error(), "Filesystem not supported") {
			t.Errorf("want 'Filesystem not supported' error, got: %v", err)
		}
	})

	t.Run("non-existing-path", func(t *testing.T) {
		if _, err := getFsType("/non/existing/path"); err == nil {
			t.Error("expected an error for a non-existing path")
		}
	})

	t.Run("empty-path", func(t *testing.T) {
		if got, err := getFsType(""); err == nil {
			t.Errorf("expected an error for an empty path, got '%s'", got)
		}
	})

	t.Run("dangling-symlink", func(t *testing.T) {
		link := filepath.Join(t.TempDir(), "dangling")
		if err := os.Symlink(filepath.Join(t.TempDir(), "missing"), link); err != nil {
			t.Fatal(err)
		}
		if got, err := getFsType(link); err == nil {
			t.Errorf("expected an error for a dangling symlink, got '%s'", got)
		}
	})
}

// a plugin name must never make xpire load code from outside of the
// plugin directory
func TestLoadPluginInvalidName(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../evil", "a/b", "/tmp/evil", "nope"} {
		t.Run("name-"+strings.ReplaceAll(name, "/", "_"), func(t *testing.T) {
			p, err := loadPlugin(name)
			if err == nil {
				t.Errorf("loadPlugin(%q) succeeded", name)
			}
			if p != nil {
				t.Errorf("loadPlugin(%q) returned a plugin", name)
			}
		})
	}
}
