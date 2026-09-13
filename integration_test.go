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
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/pkg/xattr"
)

const (
	expiredDate = "2002-01-01 15:00:00"
	futureDate  = "2099-01-01 15:00:00"
	badDate     = "205-02 111"
)

func requireRoot(t *testing.T) {
	t.Helper()
	if os.Getuid() != 0 {
		t.Skip("integration tests need root permissions")
	}
}

// testName returns a name derived from the running test that can be used
// as file or dataset name
func testName(t *testing.T) string {
	return strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
}

// sh runs a command and fails the test if it does not succeed
func sh(t *testing.T, name string, args ...string) {
	t.Helper()
	if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
}

func setExpire(t *testing.T, path, date string) {
	t.Helper()
	if err := xattr.Set(path, "user.expire", []byte(date)); err != nil {
		t.Fatal(err)
	}
}

// expireOf returns the user.expire xattr of path and whether it is set
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

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected '%s' to exist: %v", path, err)
	}
}

func assertGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected '%s' to be deleted, stat returned: %v", path, err)
	}
}

func assertNotContains(t *testing.T, out, unwanted string) {
	t.Helper()
	if strings.Contains(out, unwanted) {
		t.Errorf("output must not contain %q\n%s", unwanted, out)
	}
}

func assertRun(t *testing.T, wantRC int, wantOutput []string, args ...string) string {
	t.Helper()
	out, rc := runXpire(t, args...)
	if rc != wantRC {
		t.Errorf("xpire %s: want exit code %d, got %d\n%s", strings.Join(args, " "), wantRC, rc, out)
	}
	for _, want := range wantOutput {
		if !strings.Contains(out, want) {
			t.Errorf("xpire %s: output does not contain %q\n%s", strings.Join(args, " "), want, out)
		}
	}
	return out
}
