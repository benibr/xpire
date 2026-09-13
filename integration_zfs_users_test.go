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
)

// TestZFSUsers runs ZFS plugin tests with unprivileged users.
// Run all tests:       go test -tags integration -run TestZFSUsers
func TestZFSUsers(t *testing.T) {
	requireZFS(t)
	requireUserAccess(t, ownerUID, ownerGID, zfsMount)
	requireUserAccess(t, otherUID, otherGID, zfsMount)

	// Documents current behaviour: unlike btrfs, listing does not need root
	// permissions, see DECISION.md
	t.Run("list-as-non-root", func(t *testing.T) {
		base, path := newZfsBase(t)
		setExpire(t, newDataset(t, base, "ds"), expiredDate)
		assertRunAs(t, ownerUID, ownerGID, RC_OK, []string{"↳ Dataset '" + base + "/ds' expired since " + expiredDate},
			"--path", path, "--list")
	})

	t.Run("prune-as-non-root", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		chown(t, ds, ownerUID, ownerGID)
		setExpire(t, ds, expiredDate)
		assertRunAs(t, ownerUID, ownerGID, RC_ERR_PLUGIN, []string{"failed to destroy dataset '" + base + "/ds'"},
			"--path", ds, "--prune")
		assertDatasetExists(t, base+"/ds")
	})

	t.Run("set-as-owner", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		chown(t, ds, ownerUID, ownerGID)
		assertRunAs(t, ownerUID, ownerGID, RC_OK, nil, "--path", ds, "--set", expiredDate)
		if got, _ := expireOf(t, ds); got != expiredDate {
			t.Errorf("want user.expire %q, got %q", expiredDate, got)
		}
	})

	t.Run("unset-as-owner", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		chown(t, ds, ownerUID, ownerGID)
		setExpire(t, ds, expiredDate)
		assertRunAs(t, ownerUID, ownerGID, RC_OK, nil, "--path", ds, "--unset")
		if _, ok := expireOf(t, ds); ok {
			t.Error("user.expire is still set")
		}
	})

	t.Run("set-as-other-user", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		chown(t, ds, ownerUID, ownerGID)
		assertRunAs(t, otherUID, otherGID, RC_ERR_FS, []string{"permission denied"},
			"--path", ds, "--set", expiredDate)
		if _, ok := expireOf(t, ds); ok {
			t.Error("user.expire must not be set")
		}
	})

	t.Run("unset-as-other-user", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		chown(t, ds, ownerUID, ownerGID)
		setExpire(t, ds, futureDate)
		assertRunAs(t, otherUID, otherGID, RC_ERR_FS, []string{"permission denied"},
			"--path", ds, "--unset")
		if got, _ := expireOf(t, ds); got != futureDate {
			t.Errorf("want user.expire %q, got %q", futureDate, got)
		}
	})

	t.Run("set-below-inaccessible-directory", func(t *testing.T) {
		base, _ := newZfsBase(t)
		chmod(t, newDataset(t, base, "private"), 0700)
		ds := newDataset(t, base, "private/ds")
		chown(t, ds, ownerUID, ownerGID)
		assertRunAs(t, ownerUID, ownerGID, RC_ERR_FS, []string{"permission denied"},
			"--path", ds, "--set", expiredDate)
		if _, ok := expireOf(t, ds); ok {
			t.Error("user.expire must not be set")
		}
	})

	// Documents current behaviour: every user with write permission can set
	// an expiration date that root will act on, see DECISION.md
	t.Run("set-as-other-user-on-world-writable", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		chmod(t, ds, 0777)
		assertRunAs(t, otherUID, otherGID, RC_OK, nil, "--path", ds, "--set", expiredDate)
		if got, _ := expireOf(t, ds); got != expiredDate {
			t.Errorf("want user.expire %q, got %q", expiredDate, got)
		}
	})

	t.Run("root-prunes-date-set-by-owner", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		chown(t, ds, ownerUID, ownerGID)
		assertRunAs(t, ownerUID, ownerGID, RC_OK, nil, "--path", ds, "--set", expiredDate)
		assertRun(t, RC_OK, []string{"↳ Dataset '" + base + "/ds' expired since " + expiredDate},
			"--path", ds, "--prune")
		assertDatasetGone(t, base+"/ds")
	})
}
