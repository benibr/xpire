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
	"syscall"
	"testing"

	"github.com/moby/sys/mountinfo"
)

// mountpoint of the btrfs filesystem created by tests/setup-btrfs.sh
var btrfsMount = absPath("tests/mnt/btrfs")

func requireBtrfs(t *testing.T) {
	t.Helper()
	requireRoot(t)
	var stat syscall.Statfs_t
	if err := syscall.Statfs(btrfsMount, &stat); err != nil || stat.Type != 0x9123683E {
		t.Skipf("no btrfs filesystem mounted at '%s', run tests/setup-btrfs.sh first", btrfsMount)
	}
}

// newBtrfsBase creates an empty directory for the running test on the
// btrfs filesystem and returns its path
func newBtrfsBase(t *testing.T) string {
	t.Helper()
	base := filepath.Join(btrfsMount, testName(t))
	if err := os.Mkdir(base, 0755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(base) })
	return base
}

// newSubvolume creates a subvolume that is deleted after the test
// unless xpire already did so
func newSubvolume(t *testing.T, path string) string {
	t.Helper()
	sh(t, "btrfs", "subvolume", "create", path)
	t.Cleanup(func() {
		if _, err := os.Stat(path); err == nil {
			sh(t, "btrfs", "subvolume", "delete", path)
		}
	})
	return path
}

// TestBTRFS runs all BTRFS plugin tests.
// Run all BTRFS tests:      go test -tags integration -run TestBTRFS
// Run a single test:        go test -tags integration -run "TestBTRFS/list"
func TestBTRFS(t *testing.T) {
	requireBtrfs(t)

	t.Run("set-expire-date", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		assertRun(t, RC_OK, []string{"setting expiration date on '" + sv + "' to " + expiredDate},
			"--path", sv, "--set", expiredDate)
		if got, _ := expireOf(t, sv); got != expiredDate {
			t.Errorf("want user.expire %q, got %q", expiredDate, got)
		}
	})

	t.Run("set-expire-date-overwrites", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		setExpire(t, sv, futureDate)
		assertRun(t, RC_OK, nil, "--path", sv, "--set", expiredDate)
		if got, _ := expireOf(t, sv); got != expiredDate {
			t.Errorf("want user.expire %q, got %q", expiredDate, got)
		}
	})

	t.Run("set-invalid-date", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		assertRun(t, RC_ERR_ARGS, nil, "--path", sv, "--set", badDate)
		if _, ok := expireOf(t, sv); ok {
			t.Error("user.expire must not be set")
		}
	})

	t.Run("set-on-non-subvolume-directory", func(t *testing.T) {
		dir := newBtrfsBase(t)
		assertRun(t, RC_ERR_FS, []string{"is not a btrfs subvolume"}, "--path", dir, "--set", expiredDate)
		if _, ok := expireOf(t, dir); ok {
			t.Error("user.expire must not be set")
		}
	})

	t.Run("unset-existing-expire-date", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		setExpire(t, sv, expiredDate)
		assertRun(t, RC_OK, []string{"unsetting expiration date on '" + sv + "'"}, "--path", sv, "--unset")
		if _, ok := expireOf(t, sv); ok {
			t.Error("user.expire is still set")
		}
	})

	t.Run("unset-non-existing-expire-date", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		assertRun(t, RC_ERR_FS, []string{"failed to remove xattr"}, "--path", sv, "--unset")
	})

	t.Run("list", func(t *testing.T) {
		base := newBtrfsBase(t)
		expired := newSubvolume(t, filepath.Join(base, "expired"))
		future := newSubvolume(t, filepath.Join(base, "future"))
		none := newSubvolume(t, filepath.Join(base, "none"))
		setExpire(t, expired, expiredDate)
		setExpire(t, future, futureDate)
		abs, _ := filepath.Abs(base)
		out := assertRun(t, RC_OK, []string{
			"searching for all expire dates in '" + abs + "'",
			"↳ Subvolume '" + testName(t) + "/expired' expired since " + expiredDate,
			"↳ Subvolume '" + testName(t) + "/future' expires in " + futureDate,
		}, "--path", base, "--list")
		assertNotContains(t, out, "/none'")
		for _, sv := range []string{expired, future, none} {
			assertExists(t, sv)
		}
	})

	t.Run("prune-non-expired", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		setExpire(t, sv, futureDate)
		out := assertRun(t, RC_OK, []string{"pruning expired data in '" + sv + "'"}, "--path", sv, "--prune")
		assertNotContains(t, out, "expired since")
		assertExists(t, sv)
	})

	t.Run("prune-one-expired", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		setExpire(t, sv, expiredDate)
		assertRun(t, RC_OK, []string{"↳ Subvolume '" + testName(t) + "/sv' expired since " + expiredDate},
			"--path", sv, "--prune")
		assertGone(t, sv)
	})

	t.Run("prune-one-sub-subvolume-expired", func(t *testing.T) {
		parent := newSubvolume(t, filepath.Join(newBtrfsBase(t), "parent"))
		expired := newSubvolume(t, filepath.Join(parent, "expired"))
		future := newSubvolume(t, filepath.Join(parent, "future"))
		setExpire(t, expired, expiredDate)
		setExpire(t, future, futureDate)
		out := assertRun(t, RC_OK, []string{"↳ Subvolume '" + testName(t) + "/parent/expired' expired since " + expiredDate},
			"--path", parent, "--prune")
		assertNotContains(t, out, "/future'")
		assertGone(t, expired)
		assertExists(t, future)
		assertExists(t, parent)
	})

	t.Run("prune-on-non-subvolume-directory", func(t *testing.T) {
		base := newBtrfsBase(t)
		sv := newSubvolume(t, filepath.Join(base, "sv"))
		setExpire(t, sv, expiredDate)
		dir := filepath.Join(base, "dir")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		out := assertRun(t, RC_OK, []string{"pruning expired data in '" + dir + "'"}, "--path", dir, "--prune")
		assertNotContains(t, out, "expired since")
		assertExists(t, sv)
	})

	t.Run("prune-does-not-touch-siblings-with-same-prefix", func(t *testing.T) {
		base := newBtrfsBase(t)
		target := newSubvolume(t, filepath.Join(base, "data0"))
		sibling := newSubvolume(t, filepath.Join(base, "data01"))
		setExpire(t, sibling, expiredDate)
		out := assertRun(t, RC_OK, nil, "--path", target, "--prune")
		assertNotContains(t, out, "data01")
		assertExists(t, sibling)
	})

	t.Run("prune-expired-snapshot", func(t *testing.T) {
		base := newBtrfsBase(t)
		source := newSubvolume(t, filepath.Join(base, "source"))
		snapshot := filepath.Join(base, "snapshot")
		sh(t, "btrfs", "subvolume", "snapshot", source, snapshot)
		t.Cleanup(func() {
			if _, err := os.Stat(snapshot); err == nil {
				sh(t, "btrfs", "subvolume", "delete", snapshot)
			}
		})
		setExpire(t, snapshot, expiredDate)
		assertRun(t, RC_OK, []string{"↳ Subvolume '" + testName(t) + "/snapshot' expired since " + expiredDate},
			"--path", base, "--prune")
		assertGone(t, snapshot)
		assertExists(t, source)
	})

	t.Run("prune-wrong-time-format", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		setExpire(t, sv, badDate)
		assertRun(t, RC_OK, []string{
			"level=warning msg=cannot parse expire date format",
			`parsing time "` + badDate + `"`,
		}, "--path", sv, "--prune")
		assertExists(t, sv)
	})

	t.Run("prune-mounted-under-different-name", func(t *testing.T) {
		base := newBtrfsBase(t)
		vol := newSubvolume(t, filepath.Join(base, "vol"))
		expired := newSubvolume(t, filepath.Join(vol, "expired"))
		future := newSubvolume(t, filepath.Join(vol, "future"))
		sibling := newSubvolume(t, filepath.Join(base, "sibling"))
		setExpire(t, expired, expiredDate)
		setExpire(t, future, futureDate)
		setExpire(t, sibling, expiredDate)

		mnt := filepath.Join(base, "mnt")
		if err := os.Mkdir(mnt, 0755); err != nil {
			t.Fatal(err)
		}
		absMount, _ := filepath.Abs(btrfsMount)
		mounts, err := mountinfo.GetMounts(mountinfo.SingleEntryFilter(absMount))
		if err != nil || len(mounts) == 0 {
			t.Fatalf("cannot find device of '%s': %v", absMount, err)
		}
		sh(t, "mount", "-o", "subvol="+testName(t)+"/vol", mounts[0].Source, mnt)
		t.Cleanup(func() { sh(t, "umount", mnt) })

		out := assertRun(t, RC_OK, []string{"↳ Subvolume '" + testName(t) + "/vol/expired' expired since " + expiredDate},
			"--path", mnt, "--prune")
		assertNotContains(t, out, "sibling")
		assertGone(t, expired)
		assertExists(t, future)
		assertExists(t, sibling)
	})
}
