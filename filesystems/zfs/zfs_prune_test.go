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
	"testing"
	"time"
)

// scopeDatasets is a pool with every kind of dataset that must survive a
// prune next to the expired ones, and a second pool mounted elsewhere
var scopeDatasets = []fakeDS{
	{name: "pool", mount: "pool"},
	{name: "pool/expired", mount: "pool/expired", expire: expiredDate},
	// sibling with the same name prefix
	{name: "pool/expired2", mount: "pool/expired2", expire: expiredDate},
	{name: "pool/future", mount: "pool/future", expire: futureDate},
	{name: "pool/far-future", mount: "pool/far-future", expire: "9999-12-31 23:59:59"},
	{name: "pool/no-date", mount: "pool/no-date"},
	{name: "pool/bad-date", mount: "pool/bad-date", expire: badDate},
	{name: "pool/bad-leading-space", mount: "pool/bad-leading-space", expire: " " + expiredDate},
	{name: "pool/bad-trailing-newline", mount: "pool/bad-trailing-newline", expire: expiredDate + "\n"},
	{name: "pool/bad-date-only", mount: "pool/bad-date-only", expire: "2002-01-01"},
	{name: "pool/bad-iso", mount: "pool/bad-iso", expire: "2002-01-01T15:00:00"},
	{name: "pool/bad-zone", mount: "pool/bad-zone", expire: expiredDate + " UTC"},
	{name: "pool/unmounted", mount: "pool/unmounted", unmounted: true, expire: expiredDate},
	{name: "pool/no-mountpoint", mount: "none"},
	{name: "pool/legacy", mount: "legacy"},
	{name: "pool/sub", mount: "pool/sub"},
	{name: "pool/sub/deep", mount: "pool/sub/deep", expire: expiredDate},
	{name: "other", mount: "other"},
	{name: "other/expired", mount: "other/expired", expire: expiredDate},
}

// TestPruneExpiredFakeZfs asserts which datasets a prune destroys.
// Run all fake zfs tests:  go test ./filesystems/zfs/ -run FakeZfs
// Run a single test:       go test ./filesystems/zfs/ -run "TestPruneExpiredFakeZfs/scope"
func TestPruneExpiredFakeZfs(t *testing.T) {
	t.Run("scope", func(t *testing.T) {
		tests := []struct {
			name string
			path string
			want []string
		}{
			{name: "one-dataset", path: "pool/expired", want: []string{"pool/expired"}},
			{name: "trailing-slash", path: "pool/expired/", want: []string{"pool/expired"}},
			{name: "dot-segments", path: "pool/future/../expired", want: []string{"pool/expired"}},
			{name: "whole-pool", path: "pool", want: []string{"pool/expired", "pool/expired2", "pool/sub/deep"}},
			{name: "below-dataset-without-date", path: "pool/sub", want: []string{"pool/sub/deep"}},
			{name: "other-pool", path: "other", want: []string{"other/expired"}},
			{name: "future", path: "pool/future", want: nil},
			{name: "no-date", path: "pool/no-date", want: nil},
			{name: "bad-date", path: "pool/bad-date", want: nil},
			{name: "unmounted", path: "pool/unmounted", want: nil},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := newFakeZfs(t, scopeDatasets)
				if _, err := (ZfsPlugin{}).PruneExpired(f.path(tt.path)); err != nil {
					t.Errorf("PruneExpired failed: %v", err)
				}
				f.assertDestroyed(t, tt.want...)
			})
		}
	})

	t.Run("scope-symlink", func(t *testing.T) {
		f := newFakeZfs(t, scopeDatasets)
		link := filepath.Join(t.TempDir(), "link")
		if err := os.Symlink(f.path("pool/expired"), link); err != nil {
			t.Fatal(err)
		}
		if _, err := (ZfsPlugin{}).PruneExpired(link); err != nil {
			t.Errorf("PruneExpired failed: %v", err)
		}
		f.assertDestroyed(t, "pool/expired")
	})

	t.Run("scope-relative-path", func(t *testing.T) {
		f := newFakeZfs(t, scopeDatasets)
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(f.path("pool")); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chdir(cwd) })
		if _, err := (ZfsPlugin{}).PruneExpired("sub"); err != nil {
			t.Errorf("PruneExpired failed: %v", err)
		}
		f.assertDestroyed(t, "pool/sub/deep")
	})

	t.Run("plain-directory-inside-dataset", func(t *testing.T) {
		f := newFakeZfs(t, scopeDatasets)
		dir := f.path("pool/expired/dir")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if _, err := (ZfsPlugin{}).PruneExpired(dir); err != nil {
			t.Errorf("PruneExpired failed: %v", err)
		}
		// the expired dataset is above the given path, not below
		f.assertDestroyed(t)
	})

	t.Run("plain-directory-outside-datasets", func(t *testing.T) {
		f := newFakeZfs(t, scopeDatasets)
		(ZfsPlugin{}).PruneExpired(t.TempDir())
		f.assertDestroyed(t)
	})

	// Documents current behaviour: there is no guard against pruning '/',
	// every expired dataset of every pool on the host is in scope then.
	// See K11 in TEST_PLAN.md
	t.Run("root-path", func(t *testing.T) {
		f := newFakeZfs(t, scopeDatasets)
		if _, err := (ZfsPlugin{}).PruneExpired("/"); err != nil {
			t.Errorf("PruneExpired failed: %v", err)
		}
		f.assertDestroyed(t, "pool/expired", "pool/expired2", "pool/sub/deep", "other/expired")
	})

	// KNOWN ISSUE K1 (TEST_PLAN.md): the error of CleanPath is discarded, so
	// a path that does not exist becomes "" and every dataset of the host
	// lies "under" it
	for name, path := range map[string]string{
		"non-existing-path": "pool/does-not-exist",
		"dangling-symlink":  "dangling",
	} {
		t.Run(name, func(t *testing.T) {
			f := newFakeZfs(t, scopeDatasets)
			if err := os.Symlink(f.path("gone"), f.path("dangling")); err != nil {
				t.Fatal(err)
			}
			if _, err := (ZfsPlugin{}).PruneExpired(f.path(path)); err == nil {
				t.Error("PruneExpired succeeded on a path that cannot be resolved")
			}
			f.assertDestroyed(t)
		})
	}

	t.Run("multiple-expired-one-fails", func(t *testing.T) {
		f := newFakeZfs(t, []fakeDS{
			{name: "pool", mount: "pool"},
			{name: "pool/a", mount: "pool/a", expire: expiredDate},
			{name: "pool/b", mount: "pool/b", expire: expiredDate},
			{name: "pool/c", mount: "pool/c", expire: expiredDate},
			{name: "pool/future", mount: "pool/future", expire: futureDate},
		})
		t.Setenv("FAKE_ZFS_FAIL_DESTROY", "pool/b")
		_, err := (ZfsPlugin{}).PruneExpired(f.path("pool"))
		// the failure of one dataset must not stop the others
		f.assertDestroyed(t, "pool/a", "pool/b", "pool/c")
		if err == nil {
			t.Fatal("PruneExpired succeeded although a destroy failed")
		}
		if !strings.Contains(err.Error(), "'pool/b'") {
			t.Errorf("error does not name the failed dataset: %v", err)
		}
		for _, ds := range []string{"'pool/a'", "'pool/c'", "'pool/future'"} {
			if strings.Contains(err.Error(), ds) {
				t.Errorf("error names %s which did not fail: %v", ds, err)
			}
		}
	})

	// KNOWN ISSUE K7 (TEST_PLAN.md): errors of 'zfs get' are ignored, a
	// dataset whose properties cannot be read is skipped silently
	t.Run("get-fails", func(t *testing.T) {
		f := newFakeZfs(t, []fakeDS{
			{name: "pool", mount: "pool"},
			{name: "pool/broken", mount: "pool/broken", expire: expiredDate},
			{name: "pool/expired", mount: "pool/expired", expire: expiredDate},
		})
		t.Setenv("FAKE_ZFS_FAIL_GET", "pool/broken")
		_, err := (ZfsPlugin{}).PruneExpired(f.path("pool"))
		f.assertDestroyed(t, "pool/expired")
		if err == nil {
			t.Error("PruneExpired succeeded although a dataset could not be checked")
		}
	})

	// Documents current behaviour: snapshots and volumes have no mountpoint
	// and therefore never expire, although the README of the plugin says
	// snapshots are pruned. See K9 in TEST_PLAN.md
	t.Run("snapshots-and-volumes", func(t *testing.T) {
		f := newFakeZfs(t, []fakeDS{
			{name: "pool", mount: "pool"},
			{name: "pool/future", mount: "pool/future", expire: futureDate},
			{name: "pool/future@snap", mount: "-", typ: "snapshot"},
			{name: "pool/volume", mount: "-", typ: "volume"},
		})
		if _, err := (ZfsPlugin{}).PruneExpired(f.path("pool")); err != nil {
			t.Errorf("PruneExpired failed: %v", err)
		}
		f.assertDestroyed(t)
	})

	// an expired dataset must never take its children with it: destroy is
	// called without '-r', so zfs refuses and the error is reported
	t.Run("expired-parent-with-future-child", func(t *testing.T) {
		f := newFakeZfs(t, []fakeDS{
			{name: "pool", mount: "pool"},
			{name: "pool/parent", mount: "pool/parent", expire: expiredDate},
			{name: "pool/parent/child", mount: "pool/parent/child", expire: futureDate},
		})
		_, err := (ZfsPlugin{}).PruneExpired(f.path("pool"))
		f.assertDestroyed(t, "pool/parent")
		if err == nil || !strings.Contains(err.Error(), "'pool/parent'") {
			t.Errorf("error does not name the dataset with children: %v", err)
		}
	})

	t.Run("expired-dataset-with-snapshot", func(t *testing.T) {
		f := newFakeZfs(t, []fakeDS{
			{name: "pool", mount: "pool"},
			{name: "pool/data", mount: "pool/data", expire: expiredDate},
			{name: "pool/data@snap", mount: "-", typ: "snapshot"},
		})
		_, err := (ZfsPlugin{}).PruneExpired(f.path("pool"))
		f.assertDestroyed(t, "pool/data")
		if err == nil {
			t.Error("PruneExpired succeeded although the dataset has a snapshot")
		}
	})

	t.Run("unusual-names", func(t *testing.T) {
		f := newFakeZfs(t, []fakeDS{
			{name: "pool", mount: "pool"},
			{name: "pool/odd-_.:name", mount: "pool/with space/dätäset", expire: expiredDate},
			{name: "pool/odd-_.:name2", mount: "pool/with space/dätäset2", expire: futureDate},
			{name: "pool/-rf", mount: "pool/-rf", expire: expiredDate},
		})
		if _, err := (ZfsPlugin{}).PruneExpired(f.path("pool/with space")); err != nil {
			t.Errorf("PruneExpired failed: %v", err)
		}
		f.assertDestroyed(t, "pool/odd-_.:name")
	})

	// one hour is far enough from now to be stable without a fake clock,
	// the dates are stored in UTC like 'xpire --set' does
	t.Run("close-to-now", func(t *testing.T) {
		now := time.Now().UTC()
		f := newFakeZfs(t, []fakeDS{
			{name: "pool", mount: "pool"},
			{name: "pool/hour-ago", mount: "pool/hour-ago", expire: now.Add(-time.Hour).Format(TimeFormat)},
			{name: "pool/in-an-hour", mount: "pool/in-an-hour", expire: now.Add(time.Hour).Format(TimeFormat)},
		})
		if _, err := (ZfsPlugin{}).PruneExpired(f.path("pool")); err != nil {
			t.Errorf("PruneExpired failed: %v", err)
		}
		f.assertDestroyed(t, "pool/hour-ago")
	})
}
