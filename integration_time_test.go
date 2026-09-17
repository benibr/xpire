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
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// objectFactory prepares the running test and returns a function that
// creates subvolumes or datasets by name and returns their path
type objectFactory func(t *testing.T) func(name string) string

// TestBTRFSTime runs the expire date tests with btrfs subvolumes.
// Run a single test:        go test -tags integration -run "TestBTRFSTime/close-to-now"
func TestBTRFSTime(t *testing.T) {
	requireBtrfs(t)
	testTime(t, func(t *testing.T) func(string) string {
		base := newBtrfsBase(t)
		return func(name string) string { return newSubvolume(t, filepath.Join(base, name)) }
	})
}

// TestZFSTime runs the expire date tests with zfs datasets.
// Run a single test:       go test -tags integration -run "TestZFSTime/close-to-now"
func TestZFSTime(t *testing.T) {
	requireZFS(t)
	testTime(t, func(t *testing.T) func(string) string {
		base, _ := newZfsBase(t)
		return func(name string) string { return newDataset(t, base, name) }
	})
}

// testTime checks expire dates that are close to now. xpire has no way to
// fake its clock, so all dates keep a distance of one hour to now to be
// stable.
func testTime(t *testing.T, factory objectFactory) {
	t.Run("close-to-now", func(t *testing.T) {
		newObject := factory(t)
		now := time.Now().UTC()
		past := newObject("hour-ago")
		future := newObject("in-an-hour")
		setExpire(t, past, now.Add(-time.Hour).Format(time.DateTime))
		setExpire(t, future, now.Add(time.Hour).Format(time.DateTime))
		for _, path := range []string{past, future} {
			if out, rc := runXpireEnv(t, []string{"TZ=UTC"}, "--path", path, "--prune"); rc != RC_OK {
				t.Errorf("want exit code %d, got %d\n%s", RC_OK, rc, out)
			}
		}
		assertGone(t, past)
		assertExists(t, future)
	})

	// KNOWN ISSUE K3 (TEST_PLAN.md): dates are read as UTC but compared with
	// the local time. West of UTC data is deleted before the date the user
	// gave is reached, east of UTC it is kept too long.
	for _, zone := range []string{"UTC", "America/Los_Angeles", "Asia/Tokyo"} {
		t.Run("local-time-"+strings.ReplaceAll(zone, "/", "-"), func(t *testing.T) {
			location, err := time.LoadLocation(zone)
			if err != nil {
				t.Skipf("no timezone data for '%s': %v", zone, err)
			}
			newObject := factory(t)
			env := []string{"TZ=" + zone}
			now := time.Now().In(location)
			past := newObject("hour-ago")
			future := newObject("in-an-hour")
			for path, date := range map[string]string{
				past:   now.Add(-time.Hour).Format(time.DateTime),
				future: now.Add(time.Hour).Format(time.DateTime),
			} {
				if out, rc := runXpireEnv(t, env, "--path", path, "--set", date); rc != RC_OK {
					t.Fatalf("want exit code %d, got %d\n%s", RC_OK, rc, out)
				}
				// the date is stored like the user gave it
				if got, _ := expireOf(t, path); got != date {
					t.Errorf("want user.expire %q, got %q", date, got)
				}
				runXpireEnv(t, env, "--path", path, "--prune")
			}
			assertExists(t, future)
			assertGone(t, past)
		})
	}

	t.Run("leap-day", func(t *testing.T) {
		path := factory(t)("sv")
		assertRun(t, RC_OK, nil, "--path", path, "--set", "2024-02-29 12:00:00")
		if got, _ := expireOf(t, path); got != "2024-02-29 12:00:00" {
			t.Errorf("want user.expire %q, got %q", "2024-02-29 12:00:00", got)
		}
		assertRun(t, RC_ERR_ARGS, nil, "--path", path, "--set", "2023-02-29 12:00:00")
		if got, _ := expireOf(t, path); got != "2024-02-29 12:00:00" {
			t.Errorf("a rejected date changed user.expire to %q", got)
		}
	})

	// 02:30 does not exist on that day in Berlin, the clock jumps from 02:00
	// to 03:00. What --set accepts must be readable by --prune.
	t.Run("daylight-saving-gap", func(t *testing.T) {
		if _, err := time.LoadLocation("Europe/Berlin"); err != nil {
			t.Skipf("no timezone data for 'Europe/Berlin': %v", err)
		}
		newObject := factory(t)
		env := []string{"TZ=Europe/Berlin"}
		path := newObject("sv")
		survivor := newObject("survivor")
		setExpire(t, survivor, futureDate)
		if out, rc := runXpireEnv(t, env, "--path", path, "--set", "2026-03-29 02:30:00"); rc != RC_OK {
			t.Fatalf("want exit code %d, got %d\n%s", RC_OK, rc, out)
		}
		out, rc := runXpireEnv(t, env, "--path", path, "--prune")
		if rc != RC_OK {
			t.Errorf("want exit code %d, got %d\n%s", RC_OK, rc, out)
		}
		assertNotContains(t, out, "cannot parse")
		assertGone(t, path)
		assertExists(t, survivor)
	})
}
