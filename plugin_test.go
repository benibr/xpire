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
	"testing"

	"xpire/pluginapi"
)

func TestLoadPlugin(t *testing.T) {
	t.Run("unknown-plugin", func(t *testing.T) {
		p, err := loadPlugin("unknown")
		if err == nil {
			t.Error("expected an error for an unknown plugin")
		}
		if p != nil {
			t.Error("expected no plugin for an unknown plugin")
		}
	})

	for _, name := range []string{"btrfs", "zfs"} {
		t.Run(name, func(t *testing.T) {
			p, err := loadPlugin(name)
			if err != nil {
				t.Fatal(err)
			}
			sym, err := p.Lookup("FsPlugin")
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := sym.(pluginapi.FsPluginApi); !ok {
				t.Errorf("FsPlugin of plugin '%s' does not implement pluginapi.FsPluginApi", name)
			}
		})
	}
}
