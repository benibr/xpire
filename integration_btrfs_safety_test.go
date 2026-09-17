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
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/moby/sys/mountinfo"
	"github.com/pkg/xattr"
)

// subvolumeList returns the output of 'btrfs subvolume list' to compare
// the subvolumes of the test filesystem before and after a xpire run
func subvolumeList(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("btrfs", "subvolume", "list", btrfsMount).CombinedOutput()
	if err != nil {
		t.Fatalf("btrfs subvolume list: %v\n%s", err, out)
	}
	return string(out)
}

// TestBTRFSSafety checks that a prune deletes expired subvolumes and
// nothing else. Every test has subvolumes that must survive.
// Run all safety tests:     go test -tags integration -run TestBTRFSSafety
// Run a single test:        go test -tags integration -run "TestBTRFSSafety/prune-matrix"
func TestBTRFSSafety(t *testing.T) {
	requireBtrfs(t)

	t.Run("prune-matrix", func(t *testing.T) {
		base := newBtrfsBase(t)
		expired := newSubvolume(t, filepath.Join(base, "data"))
		survivors := map[string]string{
			"data-future":   futureDate,
			"data-no-date":  "",
			"data-bad-date": badDate,
			"data-empty":    " ",
			"data-newline":  expiredDate + "\n",
			"data-iso-date": "2002-01-01T15:00:00",
			"data2":         futureDate,
		}
		setExpire(t, expired, expiredDate)
		for name, date := range survivors {
			sv := newSubvolume(t, filepath.Join(base, name))
			if date != "" {
				setExpire(t, sv, date)
			}
		}
		dir := filepath.Join(base, "data-dir")
		file := filepath.Join(dir, "file")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
		// a date on something that is not a subvolume means nothing
		setExpire(t, dir, expiredDate)
		setExpire(t, file, expiredDate)

		assertRun(t, RC_OK, nil, "--path", base, "--prune")
		assertGone(t, expired)
		for name := range survivors {
			assertExists(t, filepath.Join(base, name))
		}
		assertExists(t, file)
	})

	t.Run("prune-multiple-expired", func(t *testing.T) {
		base := newBtrfsBase(t)
		future := newSubvolume(t, filepath.Join(base, "future"))
		setExpire(t, future, futureDate)
		var expired []string
		for _, name := range []string{"a", "b", "c"} {
			sv := newSubvolume(t, filepath.Join(base, name))
			setExpire(t, sv, expiredDate)
			expired = append(expired, sv)
		}
		assertRun(t, RC_OK, nil, "--path", base, "--prune")
		for _, sv := range expired {
			assertGone(t, sv)
		}
		assertExists(t, future)
	})

	// KNOWN ISSUE K2 (TEST_PLAN.md): btrfs refuses to delete a subvolume
	// that contains another one, but xpire exits with 0
	t.Run("prune-expired-parent-with-future-child", func(t *testing.T) {
		parent := newSubvolume(t, filepath.Join(newBtrfsBase(t), "parent"))
		child := newSubvolume(t, filepath.Join(parent, "child"))
		setExpire(t, parent, expiredDate)
		setExpire(t, child, futureDate)
		out, rc := runXpire(t, "--path", parent, "--prune")
		assertExists(t, child)
		assertExists(t, parent)
		if rc == RC_OK {
			t.Errorf("want an error exit code because the parent was not deleted, got %d\n%s", rc, out)
		}
	})

	// the order in which btrfs lists parent and child is not defined, so
	// the parent may need a second run
	t.Run("prune-expired-parent-and-child", func(t *testing.T) {
		base := newBtrfsBase(t)
		parent := newSubvolume(t, filepath.Join(base, "parent"))
		child := newSubvolume(t, filepath.Join(parent, "child"))
		sibling := newSubvolume(t, filepath.Join(base, "sibling"))
		setExpire(t, parent, expiredDate)
		setExpire(t, child, expiredDate)
		setExpire(t, sibling, futureDate)
		runXpire(t, "--path", base, "--prune")
		assertGone(t, child)
		runXpire(t, "--path", base, "--prune")
		assertGone(t, parent)
		assertExists(t, sibling)
	})

	// Documents current behaviour: only the date of the subvolume counts,
	// everything inside an expired subvolume is deleted with it
	t.Run("prune-expired-subvolume-with-files", func(t *testing.T) {
		base := newBtrfsBase(t)
		expired := newSubvolume(t, filepath.Join(base, "expired"))
		if err := os.WriteFile(filepath.Join(expired, "file"), []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
		setExpire(t, filepath.Join(expired, "file"), futureDate)
		setExpire(t, expired, expiredDate)
		assertRun(t, RC_OK, nil, "--path", base, "--prune")
		assertGone(t, expired)
	})

	t.Run("prune-scope", func(t *testing.T) {
		tests := []struct {
			name string
			// path to prune, built from the path of the subvolume 'a'
			path func(t *testing.T, a string) string
		}{
			{name: "absolute", path: func(t *testing.T, a string) string { return a }},
			{name: "trailing-slash", path: func(t *testing.T, a string) string { return a + "/" }},
			{name: "dot-segments", path: func(t *testing.T, a string) string { return a + "/../b/../a" }},
			{name: "relative", path: func(t *testing.T, a string) string {
				rel, err := filepath.Rel(binDir, a)
				if err != nil {
					t.Fatal(err)
				}
				return rel
			}},
			{name: "symlink", path: func(t *testing.T, a string) string {
				link := filepath.Join(t.TempDir(), "link")
				if err := os.Symlink(a, link); err != nil {
					t.Fatal(err)
				}
				return link
			}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				base := newBtrfsBase(t)
				a := newSubvolume(t, filepath.Join(base, "a"))
				inside := newSubvolume(t, filepath.Join(a, "inside"))
				b := newSubvolume(t, filepath.Join(base, "b"))
				ab := newSubvolume(t, filepath.Join(base, "ab"))
				for _, sv := range []string{inside, b, ab} {
					setExpire(t, sv, expiredDate)
				}
				out := assertRun(t, RC_OK, nil, "--path", tt.path(t, a), "--prune")
				assertNotContains(t, out, "/b'")
				assertGone(t, inside)
				assertExists(t, a)
				assertExists(t, b)
				assertExists(t, ab)
			})
		}
	})

	t.Run("prune-below-plain-directory", func(t *testing.T) {
		base := newBtrfsBase(t)
		dir := filepath.Join(base, "dir")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		below := newSubvolume(t, filepath.Join(dir, "sv"))
		beside := newSubvolume(t, filepath.Join(base, "sv"))
		setExpire(t, below, expiredDate)
		setExpire(t, beside, expiredDate)
		assertRun(t, RC_OK, nil, "--path", dir, "--prune")
		assertGone(t, below)
		assertExists(t, beside)
	})

	// KNOWN ISSUE K2 (TEST_PLAN.md): a failed delete is only logged, xpire
	// exits with 0 although the expired subvolume is still there
	t.Run("prune-fails", func(t *testing.T) {
		base := newBtrfsBase(t)
		locked := filepath.Join(base, "locked")
		if err := os.Mkdir(locked, 0755); err != nil {
			t.Fatal(err)
		}
		expired := newSubvolume(t, filepath.Join(locked, "expired"))
		deletable := newSubvolume(t, filepath.Join(base, "deletable"))
		setExpire(t, expired, expiredDate)
		setExpire(t, deletable, expiredDate)
		// nothing can be removed from an immutable directory
		sh(t, "chattr", "+i", locked)
		t.Cleanup(func() { sh(t, "chattr", "-i", locked) })

		out, rc := runXpire(t, "--path", base, "--prune")
		assertExists(t, expired)
		// the failure must not stop the other deletes
		assertGone(t, deletable)
		if rc == RC_OK {
			t.Errorf("want an error exit code because a delete failed, got %d\n%s", rc, out)
		}
	})

	t.Run("prune-in-subvolume-mount-keeps-outside-with-same-prefix", func(t *testing.T) {
		base := newBtrfsBase(t)
		vol := newSubvolume(t, filepath.Join(base, "vol"))
		inside := newSubvolume(t, filepath.Join(vol, "vol"))
		outside := newSubvolume(t, filepath.Join(base, "volume"))
		// same name as the one inside, but next to the mounted subvolume
		twin := newSubvolume(t, filepath.Join(base, "vol2"))
		twinInside := newSubvolume(t, filepath.Join(twin, "vol"))
		for _, sv := range []string{inside, outside, twinInside} {
			setExpire(t, sv, expiredDate)
		}

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

		assertRun(t, RC_OK, nil, "--path", mnt, "--prune")
		assertGone(t, inside)
		assertExists(t, vol)
		assertExists(t, outside)
		assertExists(t, twinInside)
	})

	// the top level subvolume of the filesystem can carry a date like any
	// other directory, it must never be tried to delete it
	t.Run("prune-expired-mount-root", func(t *testing.T) {
		base := newBtrfsBase(t)
		future := newSubvolume(t, filepath.Join(base, "future"))
		setExpire(t, future, futureDate)
		setExpire(t, btrfsMount, expiredDate)
		t.Cleanup(func() {
			if err := xattr.Remove(btrfsMount, "user.expire"); err != nil {
				t.Errorf("cannot remove user.expire from '%s': %v", btrfsMount, err)
			}
		})
		out := assertRun(t, RC_OK, nil, "--path", btrfsMount, "--prune")
		assertNotContains(t, out, "expired since")
		assertExists(t, future)
		assertExists(t, btrfsMount)
	})

	t.Run("set-and-unset-through-symlink", func(t *testing.T) {
		base := newBtrfsBase(t)
		target := newSubvolume(t, filepath.Join(base, "target"))
		sibling := newSubvolume(t, filepath.Join(base, "sibling"))
		link := filepath.Join(base, "link")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		assertRun(t, RC_OK, nil, "--path", link, "--set", futureDate)
		if got, _ := expireOf(t, target); got != futureDate {
			t.Errorf("want user.expire %q on the target, got %q", futureDate, got)
		}
		if got, ok := expireOf(t, sibling); ok {
			t.Errorf("user.expire was set to %q on the sibling", got)
		}
		if got, ok := expireOf(t, base); ok {
			t.Errorf("user.expire was set to %q on the parent directory", got)
		}
		assertRun(t, RC_OK, nil, "--path", link, "--unset")
		if got, ok := expireOf(t, target); ok {
			t.Errorf("user.expire is still set to %q", got)
		}
	})

	t.Run("set-through-symlink-to-plain-directory", func(t *testing.T) {
		base := newBtrfsBase(t)
		dir := filepath.Join(base, "dir")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(base, "link")
		if err := os.Symlink(dir, link); err != nil {
			t.Fatal(err)
		}
		assertRun(t, RC_ERR_FS, []string{"is not a btrfs subvolume"}, "--path", link, "--set", expiredDate)
		if got, ok := expireOf(t, dir); ok {
			t.Errorf("user.expire was set to %q on a plain directory", got)
		}
	})

	t.Run("unusual-names", func(t *testing.T) {
		for name, svName := range map[string]string{
			"space":        "with space",
			"unicode":      "dätä-✓",
			"leading-dash": "-rf",
			"newline":      "new\nline",
			"glob":         "*",
			"quote":        `it's "quoted"`,
		} {
			t.Run(name, func(t *testing.T) {
				base := newBtrfsBase(t)
				sv := newSubvolume(t, filepath.Join(base, svName))
				canary := newSubvolume(t, filepath.Join(base, svName+"-sibling"))
				other := newSubvolume(t, filepath.Join(base, "other"))
				setExpire(t, canary, futureDate)
				assertRun(t, RC_OK, nil, "--path", sv, "--set", expiredDate)
				if got, _ := expireOf(t, sv); got != expiredDate {
					t.Errorf("want user.expire %q, got %q", expiredDate, got)
				}
				assertRun(t, RC_OK, []string{svName + "' expired since " + expiredDate}, "--path", sv, "--list")
				assertRun(t, RC_OK, nil, "--path", sv, "--prune")
				assertGone(t, sv)
				assertExists(t, canary)
				assertExists(t, other)
			})
		}
	})

	// a read-only snapshot cannot get a date itself, it inherits the date
	// its source had when the snapshot was taken
	t.Run("prune-expired-read-only-snapshot", func(t *testing.T) {
		base := newBtrfsBase(t)
		source := newSubvolume(t, filepath.Join(base, "source"))
		snapshot := filepath.Join(base, "snapshot")
		setExpire(t, source, expiredDate)
		sh(t, "btrfs", "subvolume", "snapshot", "-r", source, snapshot)
		t.Cleanup(func() {
			if _, err := os.Stat(snapshot); err == nil {
				sh(t, "btrfs", "subvolume", "delete", snapshot)
			}
		})
		setExpire(t, source, futureDate)
		if got, _ := expireOf(t, snapshot); got != expiredDate {
			t.Fatalf("want inherited user.expire %q on the snapshot, got %q", expiredDate, got)
		}
		assertRun(t, RC_OK, nil, "--path", base, "--prune")
		assertGone(t, snapshot)
		assertExists(t, source)
	})

	t.Run("prune-twice", func(t *testing.T) {
		base := newBtrfsBase(t)
		expired := newSubvolume(t, filepath.Join(base, "expired"))
		future := newSubvolume(t, filepath.Join(base, "future"))
		setExpire(t, expired, expiredDate)
		setExpire(t, future, futureDate)
		assertRun(t, RC_OK, nil, "--path", base, "--prune")
		assertGone(t, expired)
		after := subvolumeList(t)
		out := assertRun(t, RC_OK, nil, "--path", base, "--prune")
		assertNotContains(t, out, "expired since")
		if got := subvolumeList(t); got != after {
			t.Errorf("second prune changed the subvolumes, before:\n%s\nafter:\n%s", after, got)
		}
		assertExists(t, future)
	})

	t.Run("list-is-read-only", func(t *testing.T) {
		base := newBtrfsBase(t)
		expired := newSubvolume(t, filepath.Join(base, "expired"))
		setExpire(t, expired, expiredDate)
		before := subvolumeList(t)
		assertRun(t, RC_OK, []string{"/expired' expired since " + expiredDate}, "--path", base, "--list")
		if got := subvolumeList(t); got != before {
			t.Errorf("list changed the subvolumes, before:\n%s\nafter:\n%s", before, got)
		}
		if got, _ := expireOf(t, expired); got != expiredDate {
			t.Errorf("list changed user.expire to %q", got)
		}
	})

	// Documents current behaviour: with more than one action the first of
	// --set, --unset, --prune, --list wins, the others are ignored silently.
	// A change of that order must never turn a --set or --unset into a prune.
	t.Run("set-wins-over-prune", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		setExpire(t, sv, expiredDate)
		assertRun(t, RC_OK, nil, "--path", sv, "--prune", "--set", futureDate)
		assertExists(t, sv)
		if got, _ := expireOf(t, sv); got != futureDate {
			t.Errorf("want user.expire %q, got %q", futureDate, got)
		}
	})

	t.Run("unset-wins-over-prune", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		setExpire(t, sv, expiredDate)
		assertRun(t, RC_OK, nil, "--path", sv, "--prune", "--unset")
		assertExists(t, sv)
		if got, ok := expireOf(t, sv); ok {
			t.Errorf("user.expire is still set to %q", got)
		}
	})

	t.Run("prune-wins-over-list", func(t *testing.T) {
		sv := newSubvolume(t, filepath.Join(newBtrfsBase(t), "sv"))
		setExpire(t, sv, expiredDate)
		assertRun(t, RC_OK, nil, "--path", sv, "--list", "--prune")
		assertGone(t, sv)
	})

	// KNOWN ISSUE K1 (TEST_PLAN.md): a path that does not exist is not an
	// error with an explicit plugin. Uses --list only, the scope of a
	// --prune would be the filesystem of the working directory.
	t.Run("non-existing-path-with-plugin", func(t *testing.T) {
		base := newBtrfsBase(t)
		expired := newSubvolume(t, filepath.Join(base, "expired"))
		setExpire(t, expired, expiredDate)
		out, rc := runXpire(t, "--path", filepath.Join(base, "missing"), "--plugin", "btrfs", "--list")
		if rc == RC_OK {
			t.Errorf("want an error exit code, got %d\n%s", rc, out)
		}
		assertNotContains(t, out, "↳")
	})

	// KNOWN ISSUE K5 (TEST_PLAN.md): nothing checks that the path is on
	// the filesystem of an explicitly given plugin
	t.Run("zfs-plugin-on-btrfs-path", func(t *testing.T) {
		if _, err := exec.LookPath("zfs"); err != nil {
			t.Skip("zfs command not found")
		}
		base := newBtrfsBase(t)
		out, rc := runXpire(t, "--path", base, "--plugin", "zfs", "--list")
		if rc == RC_OK {
			t.Errorf("want an error exit code, got %d\n%s", rc, out)
		}
	})
}
