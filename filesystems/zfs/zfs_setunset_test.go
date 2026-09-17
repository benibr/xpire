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
	"strings"
	"testing"
	"time"
)

// TestReadOnlyFakeZfs asserts that nothing but a prune ever destroys data
func TestReadOnlyFakeZfs(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		f := newFakeZfs(t, scopeDatasets)
		if _, err := (ZfsPlugin{}).List(f.path("pool")); err != nil {
			t.Errorf("List failed: %v", err)
		}
		f.assertDestroyed(t)
		for _, want := range []string{
			"Dataset 'pool/expired' expired since " + expiredDate,
			"Dataset 'pool/future' expires in " + futureDate,
		} {
			if !strings.Contains(f.out.String(), want) {
				t.Errorf("expected log to contain '%s', got:\n%s", want, f.out)
			}
		}
	})

	t.Run("set", func(t *testing.T) {
		f := newFakeZfs(t, scopeDatasets)
		date, _ := time.Parse(TimeFormat, expiredDate)
		if err := (ZfsPlugin{}).SetExpireDate(date, f.path("pool/future")); err != nil {
			t.Errorf("SetExpireDate failed: %v", err)
		}
		if got, _ := expireOf(t, f.path("pool/future")); got != expiredDate {
			t.Errorf("user.expire = '%s', want '%s'", got, expiredDate)
		}
		f.assertDestroyed(t)
	})

	t.Run("unset", func(t *testing.T) {
		f := newFakeZfs(t, scopeDatasets)
		if err := (ZfsPlugin{}).UnsetExpireDate(f.path("pool/expired")); err != nil {
			t.Errorf("UnsetExpireDate failed: %v", err)
		}
		if got, ok := expireOf(t, f.path("pool/expired")); ok {
			t.Errorf("user.expire is still set to '%s'", got)
		}
		f.assertDestroyed(t)
	})
}

// TestSetUnsetFakeZfs asserts that dates only end up on mounted datasets
func TestSetUnsetFakeZfs(t *testing.T) {
	date, _ := time.Parse(TimeFormat, expiredDate)
	tests := []struct {
		name string
		path string
	}{
		{name: "plain-directory", path: "pool/no-date/dir"},
		{name: "unmounted-dataset", path: "pool/unmounted"},
		{name: "non-existing-path", path: "pool/does-not-exist"},
	}
	for _, tt := range tests {
		t.Run("set-on-"+tt.name, func(t *testing.T) {
			f := newFakeZfs(t, scopeDatasets)
			if err := os.Mkdir(f.path("pool/no-date/dir"), 0755); err != nil {
				t.Fatal(err)
			}
			before, _ := expireOf(t, f.path("pool/unmounted"))
			if err := (ZfsPlugin{}).SetExpireDate(date.AddDate(1, 0, 0), f.path(tt.path)); err == nil {
				t.Error("SetExpireDate succeeded")
			}
			if got, ok := expireOf(t, f.path("pool/no-date/dir")); ok {
				t.Errorf("user.expire was set to '%s' on a plain directory", got)
			}
			if got, _ := expireOf(t, f.path("pool/unmounted")); got != before {
				t.Errorf("user.expire of the unmounted dataset changed to '%s'", got)
			}
		})
		t.Run("unset-on-"+tt.name, func(t *testing.T) {
			f := newFakeZfs(t, scopeDatasets)
			if err := os.Mkdir(f.path("pool/no-date/dir"), 0755); err != nil {
				t.Fatal(err)
			}
			setExpire(t, f.path("pool/no-date/dir"), futureDate)
			if err := (ZfsPlugin{}).UnsetExpireDate(f.path(tt.path)); err == nil {
				t.Error("UnsetExpireDate succeeded")
			}
			if _, ok := expireOf(t, f.path("pool/no-date/dir")); !ok {
				t.Error("user.expire was removed from a plain directory")
			}
			if _, ok := expireOf(t, f.path("pool/unmounted")); !ok {
				t.Error("user.expire was removed from the unmounted dataset")
			}
		})
	}
}
