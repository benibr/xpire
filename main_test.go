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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// directory with xpire and plugins as installed by 'make test-install'
var binDir = absPath("tests/bin")

// absPath returns the absolute path of a path relative to the repository
func absPath(rel string) string {
	abs, err := filepath.Abs(rel)
	if err != nil {
		panic(err)
	}
	return abs
}

func TestMain(m *testing.M) {
	if _, err := os.Stat(xpireBinary()); err != nil {
		fmt.Fprintf(os.Stderr, "%v, run 'make test-install' first\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func xpireBinary() string {
	return filepath.Join(binDir, "xpire")
}

// runXpire runs the xpire binary from the test directory, so plugins are
// found, and returns its combined output and exit code
func runXpire(t *testing.T, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(xpireBinary(), args...)
	cmd.Dir = binDir
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return string(out), exitErr.ExitCode()
	}
	if err != nil {
		t.Fatalf("cannot run xpire: %v", err)
	}
	return string(out), 0
}
