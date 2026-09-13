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
	"syscall"
	"testing"
)

// pool and mountpoint created by tests/setup-zfs.sh
const zfsPool = "xpire-pool"

var zfsMount = testMount("zfs")

func requireZFS(t *testing.T) {
	t.Helper()
	requireRoot(t)
	if _, err := exec.LookPath("zfs"); err != nil {
		t.Skip("zfs command not found")
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(zfsMount, &stat); err != nil || stat.Type != 0x2FC12FC1 {
		t.Skipf("no ZFS pool mounted at '%s', run tests/setup-zfs.sh first", zfsMount)
	}
}

// newZfsBase creates a dataset for the running test that is destroyed
// recursively after the test and returns its name and path
func newZfsBase(t *testing.T) (string, string) {
	t.Helper()
	name := zfsPool + "/" + testName(t)
	sh(t, "zfs", "create", name)
	t.Cleanup(func() { sh(t, "zfs", "destroy", "-r", name) })
	return name, filepath.Join(zfsMount, testName(t))
}

// newDataset creates a dataset below the test base dataset and returns
// its path, cleanup happens with the base dataset
func newDataset(t *testing.T, base string, name string) string {
	t.Helper()
	sh(t, "zfs", "create", base+"/"+name)
	return filepath.Join(zfsMount, testName(t), name)
}

func datasetExists(name string) bool {
	return exec.Command("zfs", "list", name).Run() == nil
}

func assertDatasetExists(t *testing.T, name string) {
	t.Helper()
	if !datasetExists(name) {
		t.Errorf("expected dataset '%s' to exist", name)
	}
}

func assertDatasetGone(t *testing.T, name string) {
	t.Helper()
	if datasetExists(name) {
		t.Errorf("expected dataset '%s' to be destroyed", name)
	}
}

// TestZFS runs all ZFS plugin tests.
// Run all ZFS tests:       go test -tags integration -run TestZFS
// Run a single test:       go test -tags integration -run "TestZFS/list"
func TestZFS(t *testing.T) {
	requireZFS(t)

	t.Run("set-expire-date", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		assertRun(t, RC_OK, []string{"setting expiration date on '" + ds + "' to " + expiredDate},
			"--path", ds, "--set", expiredDate)
		if got, _ := expireOf(t, ds); got != expiredDate {
			t.Errorf("want user.expire %q, got %q", expiredDate, got)
		}
	})

	t.Run("set-expire-date-overwrites", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		setExpire(t, ds, futureDate)
		assertRun(t, RC_OK, nil, "--path", ds, "--set", expiredDate)
		if got, _ := expireOf(t, ds); got != expiredDate {
			t.Errorf("want user.expire %q, got %q", expiredDate, got)
		}
	})

	t.Run("set-invalid-date", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		assertRun(t, RC_ERR_ARGS, nil, "--path", ds, "--set", badDate)
		if _, ok := expireOf(t, ds); ok {
			t.Error("user.expire must not be set")
		}
	})

	t.Run("set-on-non-dataset-directory", func(t *testing.T) {
		_, path := newZfsBase(t)
		dir := filepath.Join(path, "dir")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		assertRun(t, RC_ERR_FS, []string{"is not a valid, mounted ZFS dataset"}, "--path", dir, "--set", expiredDate)
		if _, ok := expireOf(t, dir); ok {
			t.Error("user.expire must not be set")
		}
	})

	t.Run("unset-existing-expire-date", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		setExpire(t, ds, expiredDate)
		assertRun(t, RC_OK, []string{"unsetting expiration date on '" + ds + "'"}, "--path", ds, "--unset")
		if _, ok := expireOf(t, ds); ok {
			t.Error("user.expire is still set")
		}
	})

	t.Run("unset-non-existing-expire-date", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		assertRun(t, RC_ERR_FS, []string{"failed to remove xattr"}, "--path", ds, "--unset")
	})

	t.Run("list", func(t *testing.T) {
		base, path := newZfsBase(t)
		setExpire(t, newDataset(t, base, "expired"), expiredDate)
		setExpire(t, newDataset(t, base, "future"), futureDate)
		newDataset(t, base, "none")
		out := assertRun(t, RC_OK, []string{
			"Listing data in '" + path + "'",
			"↳ Dataset '" + base + "/expired' expired since " + expiredDate,
			"↳ Dataset '" + base + "/future' expires in " + futureDate,
		}, "--path", path, "--list")
		assertNotContains(t, out, "/none'")
		for _, ds := range []string{"expired", "future", "none"} {
			assertDatasetExists(t, base+"/"+ds)
		}
	})

	t.Run("prune-non-expired", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		setExpire(t, ds, futureDate)
		out := assertRun(t, RC_OK, []string{"pruning expired data in '" + ds + "'"}, "--path", ds, "--prune")
		assertNotContains(t, out, "expired since")
		assertDatasetExists(t, base+"/ds")
	})

	t.Run("prune-one-expired", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		setExpire(t, ds, expiredDate)
		assertRun(t, RC_OK, []string{"↳ Dataset '" + base + "/ds' expired since " + expiredDate},
			"--path", ds, "--prune")
		assertDatasetGone(t, base+"/ds")
	})

	t.Run("prune-one-sub-dataset-expired", func(t *testing.T) {
		base, _ := newZfsBase(t)
		parent := newDataset(t, base, "parent")
		setExpire(t, newDataset(t, base, "parent/expired"), expiredDate)
		setExpire(t, newDataset(t, base, "parent/future"), futureDate)
		out := assertRun(t, RC_OK, []string{"↳ Dataset '" + base + "/parent/expired' expired since " + expiredDate},
			"--path", parent, "--prune")
		assertNotContains(t, out, "/future'")
		assertDatasetGone(t, base+"/parent/expired")
		assertDatasetExists(t, base+"/parent/future")
		assertDatasetExists(t, base+"/parent")
	})

	t.Run("prune-on-non-dataset-directory", func(t *testing.T) {
		base, path := newZfsBase(t)
		setExpire(t, newDataset(t, base, "ds"), expiredDate)
		dir := filepath.Join(path, "dir")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		out := assertRun(t, RC_OK, []string{"pruning expired data in '" + dir + "'"}, "--path", dir, "--prune")
		assertNotContains(t, out, "expired since")
		assertDatasetExists(t, base+"/ds")
	})

	t.Run("prune-does-not-touch-siblings-with-same-prefix", func(t *testing.T) {
		base, _ := newZfsBase(t)
		target := newDataset(t, base, "data0")
		setExpire(t, newDataset(t, base, "data01"), expiredDate)
		out := assertRun(t, RC_OK, nil, "--path", target, "--prune")
		assertNotContains(t, out, "data01")
		assertDatasetExists(t, base+"/data01")
	})

	t.Run("prune-wrong-time-format", func(t *testing.T) {
		base, _ := newZfsBase(t)
		ds := newDataset(t, base, "ds")
		setExpire(t, ds, badDate)
		assertRun(t, RC_OK, []string{
			"level=warning msg=cannot parse expire date format",
			`parsing time "` + badDate + `"`,
		}, "--path", ds, "--prune")
		assertDatasetExists(t, base+"/ds")
	})
}
