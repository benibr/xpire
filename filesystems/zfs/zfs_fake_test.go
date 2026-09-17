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
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"testing"

	"github.com/pkg/xattr"
	"github.com/sirupsen/logrus"
)

// directory of the fake 'zfs' command, relative to this package
const fakeZfsDir = "../../tests/fake-zfs"

// dates far away from now, there is no way to fake the clock of the plugin
const (
	expiredDate = "2002-01-01 15:00:00"
	futureDate  = "2099-01-01 15:00:00"
	badDate     = "205-02 111"
)

// fakeDS describes one dataset known to the fake 'zfs' command
type fakeDS struct {
	name string
	// mountpoint relative to the root of the fake, its directory is created.
	// "none", "legacy" and "-" are used verbatim and get no directory.
	mount     string
	unmounted bool
	// dataset type, defaults to "filesystem"
	typ string
	// value of the user.expire xattr on the mountpoint, empty for no xattr
	expire string
}

type fakeZfs struct {
	root string
	log  string
	// everything the plugin logged
	out *bytes.Buffer
}

// newFakeZfs replaces the 'zfs' command by tests/fake-zfs/zfs for the
// running test and creates a mountpoint directory for every dataset.
// PATH only contains the fake afterwards, so the real 'zfs' command can
// never be reached, not even if xpire wrongly widens its scope.
// Not usable in parallel tests because PATH is process global.
func newFakeZfs(t *testing.T, datasets []fakeDS) *fakeZfs {
	t.Helper()
	fakeDir, err := filepath.Abs(fakeZfsDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeDir)
	if got, err := exec.LookPath("zfs"); err != nil || got != filepath.Join(fakeDir, "zfs") {
		t.Fatalf("fake zfs is not the 'zfs' in PATH: '%s', %v", got, err)
	}

	// CleanPath resolves symlinks, so the mountpoints must be resolved too
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeZfs{root: root, log: filepath.Join(t.TempDir(), "calls.log"), out: &bytes.Buffer{}}

	var table strings.Builder
	for _, ds := range datasets {
		mountpoint := ds.mount
		if mountpoint != "none" && mountpoint != "legacy" && mountpoint != "-" {
			mountpoint = f.path(ds.mount)
			if err := os.MkdirAll(mountpoint, 0755); err != nil {
				t.Fatal(err)
			}
			if ds.expire != "" {
				setExpire(t, mountpoint, ds.expire)
			}
		}
		mounted := "yes"
		if ds.unmounted {
			mounted = "no"
		}
		typ := ds.typ
		if typ == "" {
			typ = "filesystem"
		}
		table.WriteString(strings.Join([]string{ds.name, mountpoint, mounted, typ}, "\t") + "\n")
	}
	tablePath := filepath.Join(filepath.Dir(f.log), "datasets.tsv")
	if err := os.WriteFile(tablePath, []byte(table.String()), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.log, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_ZFS_DATASETS", tablePath)
	t.Setenv("FAKE_ZFS_LOG", f.log)
	t.Setenv("FAKE_ZFS_FAIL_DESTROY", "")
	t.Setenv("FAKE_ZFS_FAIL_GET", "")

	logger := logrus.New()
	logger.SetOutput(f.out)
	logger.SetLevel(logrus.DebugLevel)
	if err := (ZfsPlugin{}).InitLogger(logger); err != nil {
		t.Fatal(err)
	}
	return f
}

// path returns the absolute path of a path relative to the root of the fake
func (f *fakeZfs) path(rel string) string {
	return filepath.Join(f.root, rel)
}

// calls returns the arguments of every call of the fake 'zfs' command
func (f *fakeZfs) calls(t *testing.T) [][]string {
	t.Helper()
	data, err := os.ReadFile(f.log)
	if err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if line != "" {
			calls = append(calls, strings.Split(line, "\t"))
		}
	}
	return calls
}

// assertDestroyed fails unless 'zfs destroy' was called exactly once for
// each of the wanted datasets and for nothing else. It also fails on
// destroy calls with flags like '-r' and on calls the fake does not know.
func (f *fakeZfs) assertDestroyed(t *testing.T, want ...string) {
	t.Helper()
	got := []string{}
	for _, call := range f.calls(t) {
		switch {
		case call[0] == "UNEXPECTED":
			t.Errorf("unexpected call of the zfs command, see the call before in %v", f.calls(t))
		case call[0] == "destroy" && len(call) == 2:
			got = append(got, call[1])
		case call[0] == "destroy":
			t.Errorf("zfs destroy called with flags: %v", call)
		}
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, append([]string{}, want...)) {
		t.Errorf("destroyed datasets = %q, want %q", got, want)
	}
}

// setExpire sets the user.expire xattr and skips the test if the
// filesystem of the temporary directory has no user xattr support
func setExpire(t *testing.T, path string, date string) {
	t.Helper()
	err := xattr.Set(path, "user.expire", []byte(date))
	if errors.Is(err, syscall.ENOTSUP) {
		t.Skipf("no user xattr support on '%s', set TMPDIR to a filesystem that has it", path)
	}
	if err != nil {
		t.Fatal(err)
	}
}

// expireOf returns the user.expire xattr and whether it is set
func expireOf(t *testing.T, path string) (string, bool) {
	t.Helper()
	value, err := xattr.Get(path, "user.expire")
	if errors.Is(err, xattr.ENOATTR) {
		return "", false
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(value), true
}

// TestFakeZfs checks the fake itself: a prune of one expired dataset
// Run all fake zfs tests:  go test ./filesystems/zfs/ -run FakeZfs
func TestFakeZfs(t *testing.T) {
	f := newFakeZfs(t, []fakeDS{
		{name: "pool", mount: "pool"},
		{name: "pool/expired", mount: "pool/expired", expire: expiredDate},
		{name: "pool/future", mount: "pool/future", expire: futureDate},
	})
	if _, err := (ZfsPlugin{}).PruneExpired(f.path("pool")); err != nil {
		t.Errorf("PruneExpired failed: %v", err)
	}
	f.assertDestroyed(t, "pool/expired")
}
