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

// chown changes the owner of path and fails the test on errors
func chown(t *testing.T, path string, uid, gid uint32) {
	t.Helper()
	if err := os.Chown(path, int(uid), int(gid)); err != nil {
		t.Fatal(err)
	}
}

// chmod changes the permissions of path and fails the test on errors
func chmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

// TestBTRFSUsers runs BTRFS plugin tests with unprivileged users.
// Run all tests:       go test -tags integration -run TestBTRFSUsers
func TestBTRFSUsers(t *testing.T) {
	requireBtrfs(t)
	requireUserAccess(t, ownerUID, ownerGID, btrfsMount)
	requireUserAccess(t, otherUID, otherGID, btrfsMount)

	t.Run("list-as-non-root", func(t *testing.T) {
		base := newBtrfsBase(t)
		setExpire(t, newSubvolume(t, filepath.Join(base, "sv")), expiredDate)
		out := assertRunAs(t, ownerUID, ownerGID, RC_ERR_PLUGIN, []string{"needs root permissions"},
			"--path", base, "--list")
		assertNotContains(t, out, "expired since")
	})

	t.Run("prune-as-non-root", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		chown(t, sv, ownerUID, ownerGID)
		setExpire(t, sv, expiredDate)
		assertRunAs(t, ownerUID, ownerGID, RC_ERR_PLUGIN, []string{"needs root permissions"},
			"--path", sv, "--prune")
		assertExists(t, sv)
	})

	t.Run("set-as-owner", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		chown(t, sv, ownerUID, ownerGID)
		assertRunAs(t, ownerUID, ownerGID, RC_OK, nil, "--path", sv, "--set", expiredDate)
		if got, _ := expireOf(t, sv); got != expiredDate {
			t.Errorf("want user.expire %q, got %q", expiredDate, got)
		}
	})

	t.Run("unset-as-owner", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		chown(t, sv, ownerUID, ownerGID)
		setExpire(t, sv, expiredDate)
		assertRunAs(t, ownerUID, ownerGID, RC_OK, nil, "--path", sv, "--unset")
		if _, ok := expireOf(t, sv); ok {
			t.Error("user.expire is still set")
		}
	})

	t.Run("set-as-other-user", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		chown(t, sv, ownerUID, ownerGID)
		assertRunAs(t, otherUID, otherGID, RC_ERR_FS, []string{"permission denied"},
			"--path", sv, "--set", expiredDate)
		if _, ok := expireOf(t, sv); ok {
			t.Error("user.expire must not be set")
		}
	})

	t.Run("unset-as-other-user", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		chown(t, sv, ownerUID, ownerGID)
		setExpire(t, sv, futureDate)
		assertRunAs(t, otherUID, otherGID, RC_ERR_FS, []string{"permission denied"},
			"--path", sv, "--unset")
		if got, _ := expireOf(t, sv); got != futureDate {
			t.Errorf("want user.expire %q, got %q", futureDate, got)
		}
	})

	t.Run("set-below-inaccessible-directory", func(t *testing.T) {
		private := filepath.Join(newBtrfsBase(t), "private")
		if err := os.Mkdir(private, 0700); err != nil {
			t.Fatal(err)
		}
		sv := newSubvolume(t, filepath.Join(private, "sv"))
		chown(t, sv, ownerUID, ownerGID)
		assertRunAs(t, ownerUID, ownerGID, RC_ERR_FS, []string{"permission denied"},
			"--path", sv, "--set", expiredDate)
		if _, ok := expireOf(t, sv); ok {
			t.Error("user.expire must not be set")
		}
	})

	// Documents current behaviour: every user with write permission can set
	// an expiration date that root will act on, see DECISION.md
	t.Run("set-as-other-user-on-world-writable", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		chmod(t, sv, 0777)
		assertRunAs(t, otherUID, otherGID, RC_OK, nil, "--path", sv, "--set", expiredDate)
		if got, _ := expireOf(t, sv); got != expiredDate {
			t.Errorf("want user.expire %q, got %q", expiredDate, got)
		}
	})

	t.Run("root-prunes-date-set-by-owner", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		chown(t, sv, ownerUID, ownerGID)
		abs, _ := filepath.Abs(sv)
		assertRunAs(t, ownerUID, ownerGID, RC_OK, nil, "--path", sv, "--set", expiredDate)
		assertRun(t, RC_OK, []string{"↳ '" + abs + "' expired since " + expiredDate},
			"--path", sv, "--prune")
		assertGone(t, sv)
	})
}
