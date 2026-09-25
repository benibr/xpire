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
)

// datasetList returns all datasets and snapshots of the test pool to
// compare them before and after a xpire run
func datasetList(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("zfs", "list", "-H", "-o", "name", "-r", "-t", "all", zfsPool).CombinedOutput()
	if err != nil {
		t.Fatalf("zfs list: %v\n%s", err, out)
	}
	return string(out)
}

// TestZFSSafety checks that a prune destroys expired datasets and nothing
// else. Every test has datasets that must survive. All prunes stay below
// the mountpoint of the test pool, because the zfs plugin looks at the
// datasets of all pools of the host.
// Run all safety tests:    go test -tags integration -run TestZFSSafety
// Run a single test:       go test -tags integration -run "TestZFSSafety/prune-matrix"
func TestZFSSafety(t *testing.T) {
	requireZFS(t)

	t.Run("prune-matrix", func(t *testing.T) {
		base, basePath := newZfsBase(t)
		survivors := map[string]string{
			"data-future":   futureDate,
			"data-no-date":  "",
			"data-bad-date": badDate,
			"data-empty":    " ",
			"data-newline":  expiredDate + "\n",
			"data-iso-date": "2002-01-01T15:00:00",
			"data2":         futureDate,
		}
		setExpire(t, newDataset(t, base, "data"), expiredDate)
		for name, date := range survivors {
			ds := newDataset(t, base, name)
			if date != "" {
				setExpire(t, ds, date)
			}
		}
		dir := filepath.Join(basePath, "data-dir")
		file := filepath.Join(dir, "file")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
		// a date on something that is not a dataset means nothing
		setExpire(t, dir, expiredDate)
		setExpire(t, file, expiredDate)

		assertRun(t, RC_OK, nil, "--path", basePath, "--prune")
		assertDatasetGone(t, base+"/data")
		for name := range survivors {
			assertDatasetExists(t, base+"/"+name)
		}
		assertExists(t, file)
	})

	t.Run("prune-multiple-expired", func(t *testing.T) {
		base, basePath := newZfsBase(t)
		setExpire(t, newDataset(t, base, "future"), futureDate)
		for _, name := range []string{"a", "b", "c"} {
			setExpire(t, newDataset(t, base, name), expiredDate)
		}
		assertRun(t, RC_OK, nil, "--path", basePath, "--prune")
		for _, name := range []string{"a", "b", "c"} {
			assertDatasetGone(t, base+"/"+name)
		}
		assertDatasetExists(t, base+"/future")
	})

	// an expired dataset must never take its children with it
	t.Run("prune-expired-parent-with-future-child", func(t *testing.T) {
		base, _ := newZfsBase(t)
		parent := newDataset(t, base, "parent")
		child := newDataset(t, base, "parent/child")
		if err := os.WriteFile(filepath.Join(child, "file"), []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
		setExpire(t, parent, expiredDate)
		setExpire(t, child, futureDate)
		assertRun(t, RC_ERR_PLUGIN, []string{"failed to destroy dataset mounted under '" + parent + "'"},
			"--path", parent, "--prune")
		assertDatasetExists(t, base+"/parent")
		assertDatasetExists(t, base+"/parent/child")
		assertExists(t, filepath.Join(child, "file"))
	})

	// zfs lists the parent first, xpire sorts children first so that one
	// run destroys both
	t.Run("prune-expired-parent-and-child", func(t *testing.T) {
		base, basePath := newZfsBase(t)
		setExpire(t, newDataset(t, base, "parent"), expiredDate)
		setExpire(t, newDataset(t, base, "parent/child"), expiredDate)
		setExpire(t, newDataset(t, base, "sibling"), futureDate)
		assertRun(t, RC_OK, nil, "--path", basePath, "--prune")
		assertDatasetGone(t, base+"/parent/child")
		assertDatasetGone(t, base+"/parent")
		assertDatasetExists(t, base+"/sibling")
	})

	t.Run("prune-expired-dataset-with-snapshot", func(t *testing.T) {
		base, basePath := newZfsBase(t)
		data := newDataset(t, base, "data")
		setExpire(t, data, expiredDate)
		sh(t, "zfs", "snapshot", base+"/data@snap")
		assertRun(t, RC_ERR_PLUGIN, []string{"failed to destroy dataset mounted under '" + data + "'"},
			"--path", basePath, "--prune")
		assertDatasetExists(t, base+"/data")
		assertDatasetExists(t, base+"/data@snap")
	})

	// Documents current behaviour: snapshots have no mountpoint and never
	// expire, although the README of the plugin says they are pruned.
	// See K9 in TEST_PLAN.md
	t.Run("prune-keeps-snapshots", func(t *testing.T) {
		base, basePath := newZfsBase(t)
		data := newDataset(t, base, "data")
		setExpire(t, data, expiredDate)
		sh(t, "zfs", "snapshot", base+"/data@snap")
		setExpire(t, data, futureDate)
		out := assertRun(t, RC_OK, nil, "--path", basePath, "--prune")
		assertNotContains(t, out, "expired since")
		assertDatasetExists(t, base+"/data@snap")
		assertDatasetExists(t, base+"/data")
	})

	t.Run("prune-unmounted", func(t *testing.T) {
		base, basePath := newZfsBase(t)
		setExpire(t, newDataset(t, base, "data"), expiredDate)
		sh(t, "zfs", "unmount", base+"/data")
		out := assertRun(t, RC_OK, nil, "--path", basePath, "--prune")
		assertNotContains(t, out, "expired since")
		assertDatasetExists(t, base+"/data")
	})

	// the mountpoint decides if a dataset is in scope, not its name
	t.Run("prune-dataset-mounted-elsewhere", func(t *testing.T) {
		base, basePath := newZfsBase(t)
		alt := filepath.Join(testDir(), "mnt", "zfs-alt-"+testName(t))
		sh(t, "zfs", "create", "-o", "mountpoint="+alt, base+"/elsewhere")
		t.Cleanup(func() { os.Remove(alt) })
		setExpire(t, alt, expiredDate)
		setExpire(t, newDataset(t, base, "future"), futureDate)

		out := assertRun(t, RC_OK, nil, "--path", basePath, "--prune")
		assertNotContains(t, out, "expired since")
		assertDatasetExists(t, base+"/elsewhere")

		assertRun(t, RC_OK, nil, "--path", alt, "--prune")
		assertDatasetGone(t, base+"/elsewhere")
		assertDatasetExists(t, base+"/future")
	})

	t.Run("prune-scope", func(t *testing.T) {
		tests := []struct {
			name string
			// path to prune, built from the path of the dataset 'a'
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
				base, _ := newZfsBase(t)
				a := newDataset(t, base, "a")
				setExpire(t, newDataset(t, base, "a/inside"), expiredDate)
				setExpire(t, newDataset(t, base, "b"), expiredDate)
				setExpire(t, newDataset(t, base, "ab"), expiredDate)
				out := assertRun(t, RC_OK, nil, "--path", tt.path(t, a), "--prune")
				assertNotContains(t, out, "/b'")
				assertDatasetGone(t, base+"/a/inside")
				assertDatasetExists(t, base+"/a")
				assertDatasetExists(t, base+"/b")
				assertDatasetExists(t, base+"/ab")
			})
		}
	})

	t.Run("set-and-unset-through-symlink", func(t *testing.T) {
		base, basePath := newZfsBase(t)
		target := newDataset(t, base, "target")
		sibling := newDataset(t, base, "sibling")
		link := filepath.Join(basePath, "link")
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
		if got, ok := expireOf(t, basePath); ok {
			t.Errorf("user.expire was set to %q on the parent dataset", got)
		}
		assertRun(t, RC_OK, nil, "--path", link, "--unset")
		if got, ok := expireOf(t, target); ok {
			t.Errorf("user.expire is still set to %q", got)
		}
	})

	t.Run("set-through-symlink-to-plain-directory", func(t *testing.T) {
		_, basePath := newZfsBase(t)
		dir := filepath.Join(basePath, "dir")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(basePath, "link")
		if err := os.Symlink(dir, link); err != nil {
			t.Fatal(err)
		}
		assertRun(t, RC_ERR_FS, []string{"is not a valid, mounted ZFS dataset"}, "--path", link, "--set", expiredDate)
		if got, ok := expireOf(t, dir); ok {
			t.Errorf("user.expire was set to %q on a plain directory", got)
		}
	})

	t.Run("unusual-names", func(t *testing.T) {
		for name, dsName := range map[string]string{
			"leading-dash": "-rf",
			"punctuation":  "a_b.c:d-e",
			// KNOWN ISSUE K14 (TEST_PLAN.md): go-zfs splits the output of zfs
			// at spaces, xpire does not see datasets with a space in the name
			"space": "with space",
		} {
			t.Run(name, func(t *testing.T) {
				base, _ := newZfsBase(t)
				ds := newDataset(t, base, dsName)
				// same name up to the space, which separates fields in zfs output
				setExpire(t, newDataset(t, base, "with"), futureDate)
				setExpire(t, newDataset(t, base, dsName+"-sibling"), futureDate)
				assertRun(t, RC_OK, nil, "--path", ds, "--set", expiredDate)
				if got, _ := expireOf(t, ds); got != expiredDate {
					t.Errorf("want user.expire %q, got %q", expiredDate, got)
				}
				assertRun(t, RC_OK, []string{dsName + "' expired since " + expiredDate}, "--path", ds, "--list")
				assertRun(t, RC_OK, nil, "--path", ds, "--prune")
				assertDatasetGone(t, base+"/"+dsName)
				assertDatasetExists(t, base+"/with")
				assertDatasetExists(t, base+"/"+dsName+"-sibling")
			})
		}
	})

	t.Run("prune-twice", func(t *testing.T) {
		base, basePath := newZfsBase(t)
		setExpire(t, newDataset(t, base, "expired"), expiredDate)
		setExpire(t, newDataset(t, base, "future"), futureDate)
		assertRun(t, RC_OK, nil, "--path", basePath, "--prune")
		assertDatasetGone(t, base+"/expired")
		after := datasetList(t)
		out := assertRun(t, RC_OK, nil, "--path", basePath, "--prune")
		assertNotContains(t, out, "expired since")
		if got := datasetList(t); got != after {
			t.Errorf("second prune changed the datasets, before:\n%s\nafter:\n%s", after, got)
		}
	})

	t.Run("list-is-read-only", func(t *testing.T) {
		base, basePath := newZfsBase(t)
		expired := newDataset(t, base, "expired")
		setExpire(t, expired, expiredDate)
		before := datasetList(t)
		assertRun(t, RC_OK, []string{"/expired' expired since " + expiredDate}, "--path", basePath, "--list")
		if got := datasetList(t); got != before {
			t.Errorf("list changed the datasets, before:\n%s\nafter:\n%s", before, got)
		}
		if got, _ := expireOf(t, expired); got != expiredDate {
			t.Errorf("list changed user.expire to %q", got)
		}
	})

	// Documents current behaviour: with more than one action the first of
	// --set, --unset, --prune, --list wins, the others are ignored silently.
	// A change of that order must never turn a --set or --unset into a prune.
	t.Run("set-wins-over-prune", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		setExpire(t, ds, expiredDate)
		assertRun(t, RC_OK, nil, "--path", ds, "--prune", "--set", futureDate)
		assertDatasetExists(t, base+"/ds")
		if got, _ := expireOf(t, ds); got != futureDate {
			t.Errorf("want user.expire %q, got %q", futureDate, got)
		}
	})

	t.Run("unset-wins-over-prune", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		setExpire(t, ds, expiredDate)
		assertRun(t, RC_OK, nil, "--path", ds, "--prune", "--unset")
		assertDatasetExists(t, base+"/ds")
		if got, ok := expireOf(t, ds); ok {
			t.Errorf("user.expire is still set to %q", got)
		}
	})

	t.Run("prune-wins-over-list", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		setExpire(t, ds, expiredDate)
		assertRun(t, RC_OK, nil, "--path", ds, "--list", "--prune")
		assertDatasetGone(t, base+"/ds")
	})

	// KNOWN ISSUE K1 (TEST_PLAN.md): a path that does not exist is not an
	// error with an explicit plugin, all datasets of the host are in scope
	// then. Uses --list only, a --prune could destroy data of the host that
	// runs the tests. The prune is tested with the fake zfs command, see
	// filesystems/zfs/zfs_prune_test.go
	t.Run("non-existing-path-with-plugin", func(t *testing.T) {
		base, basePath := newZfsBase(t)
		setExpire(t, newDataset(t, base, "expired"), expiredDate)
		out, rc := runXpire(t, "--path", filepath.Join(basePath, "missing"), "--plugin", "zfs", "--list")
		if rc == RC_OK {
			t.Errorf("want an error exit code, got %d\n%s", rc, out)
		}
		assertNotContains(t, out, "↳")
	})

	// see K5 in TEST_PLAN.md
	t.Run("btrfs-plugin-on-zfs-path", func(t *testing.T) {
		base, basePath := newZfsBase(t)
		setExpire(t, newDataset(t, base, "expired"), expiredDate)
		out, rc := runXpire(t, "--path", basePath, "--plugin", "btrfs", "--prune")
		if rc == RC_OK {
			t.Errorf("want an error exit code, got %d\n%s", rc, out)
		}
		assertDatasetExists(t, base+"/expired")
	})
}
