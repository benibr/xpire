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
	"os"
	"path/filepath"
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
		assertRunAs(t, ownerUID, ownerGID, RC_OK, []string{"↳ '" + path + "/ds' expired since " + expiredDate},
			"--path", path, "--list")
	})

	t.Run("prune-as-non-root", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		chown(t, ds, ownerUID, ownerGID)
		setExpire(t, ds, expiredDate)
		assertRunAs(t, ownerUID, ownerGID, RC_ERR_PLUGIN, []string{"failed to destroy dataset mounted under '" + ds + "'"},
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
		assertRun(t, RC_OK, []string{"↳ Dataset '" + ds + "' expired since " + expiredDate},
			"--path", ds, "--prune")
		assertDatasetGone(t, base+"/ds")
	})

	// Documents current behaviour: root trusts every date, also one that a
	// user without an account set on a dataset of somebody else, because
	// it is world-writable. Owner decision.
	t.Run("root-prunes-date-set-by-other-user-on-world-writable", func(t *testing.T) {
		base, basePath := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		private := newDataset(t, base, "private")
		chown(t, ds, ownerUID, ownerGID)
		chown(t, private, ownerUID, ownerGID)
		chmod(t, ds, 0777)
		assertRunAs(t, otherUID, otherGID, RC_OK, nil, "--path", ds, "--set", expiredDate)
		assertRunAs(t, otherUID, otherGID, RC_ERR_FS, nil, "--path", private, "--set", expiredDate)
		assertRun(t, RC_OK, nil, "--path", basePath, "--prune")
		assertDatasetGone(t, base+"/ds")
		assertDatasetExists(t, base+"/private")
	})

	// a symlink of a user must not give him more rights on its target
	t.Run("set-as-other-user-through-own-symlink", func(t *testing.T) {
		base, basePath := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		chown(t, ds, ownerUID, ownerGID)
		dir := filepath.Join(basePath, "dir")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, "link")
		if err := os.Symlink(ds, link); err != nil {
			t.Fatal(err)
		}
		chown(t, dir, otherUID, otherGID)
		if err := os.Lchown(link, int(otherUID), int(otherGID)); err != nil {
			t.Fatal(err)
		}
		assertRunAs(t, otherUID, otherGID, RC_ERR_FS, []string{"permission denied"},
			"--path", link, "--set", expiredDate)
		if got, ok := expireOf(t, ds); ok {
			t.Errorf("user.expire was set to %q", got)
		}
		assertRun(t, RC_OK, nil, "--path", basePath, "--prune")
		assertDatasetExists(t, base+"/ds")
	})
}
