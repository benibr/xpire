package helpers

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCleanPath(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	odd := filepath.Join(dir, "with space\nand newline")
	if err := os.Mkdir(odd, 0755); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(wd) })

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "absolute", path: target, want: target},
		{name: "relative", path: "target", want: target},
		{name: "dot-segments", path: "./target/../target/", want: target},
		{name: "symlink", path: link, want: target},
		{name: "path-below-symlink", path: "link/.", want: target},
		{name: "trailing-slashes", path: target + "//", want: target},
		{name: "parent-of-symlink-target", path: "link/..", want: dir},
		{name: "space-and-newline", path: odd, want: odd},
		// Documents current behaviour: an empty path is the working directory
		{name: "empty", path: "", want: dir},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CleanPath(tt.path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("CleanPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}

	t.Run("non-existing", func(t *testing.T) {
		if _, err := CleanPath(filepath.Join(dir, "missing")); err == nil {
			t.Error("expected an error for a non-existing path")
		}
	})

	t.Run("dangling-symlink", func(t *testing.T) {
		dangling := filepath.Join(dir, "dangling")
		if err := os.Symlink(filepath.Join(dir, "missing"), dangling); err != nil {
			t.Fatal(err)
		}
		if got, err := CleanPath(dangling); err == nil {
			t.Errorf("expected an error for a dangling symlink, got %q", got)
		}
	})

	t.Run("symlink-loop", func(t *testing.T) {
		loop := filepath.Join(dir, "loop")
		if err := os.Symlink(loop, loop); err != nil {
			t.Fatal(err)
		}
		if got, err := CleanPath(loop); err == nil {
			t.Errorf("expected an error for a symlink loop, got %q", got)
		}
	})
}

func TestSortPathsHierarchically(t *testing.T) {
	now := time.Now()
	paths := map[string]time.Time{
		"/a":     now,
		"/a/b/c": now,
		"/z":     now,
		"/ab":    now,
		"/a/b":   now,
	}
	got := SortPathsHierarchically(paths)
	if len(got) != len(paths) {
		t.Fatalf("SortPathsHierarchically() returned %d paths, want %d", len(got), len(paths))
	}
	index := make(map[string]int, len(got))
	for i, p := range got {
		index[p] = i
	}
	for _, tt := range []struct{ child, parent string }{
		{"/a/b/c", "/a/b"},
		{"/a/b", "/a"},
		{"/a/b/c", "/a"},
	} {
		if index[tt.child] > index[tt.parent] {
			t.Errorf("SortPathsHierarchically() = %v, want %q before %q", got, tt.child, tt.parent)
		}
	}
}

func TestIsRoot(t *testing.T) {
	if got, want := IsRoot(), os.Getuid() == 0; got != want {
		t.Errorf("IsRoot() = %v, want %v", got, want)
	}
}

func TestFindParentMount(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		got, err := FindParentMount("/")
		if err != nil {
			t.Fatal(err)
		}
		if got != "/" {
			t.Errorf("FindParentMount(\"/\") = %q, want \"/\"", got)
		}
	})

	t.Run("directory", func(t *testing.T) {
		dir, err := CleanPath(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		got, err := FindParentMount(dir)
		if err != nil {
			t.Fatal(err)
		}
		if got != "/" && got != dir && !strings.HasPrefix(dir, got+"/") {
			t.Errorf("FindParentMount(%q) = %q, which is not a parent of the path", dir, got)
		}
	})

	t.Run("non-existing-path", func(t *testing.T) {
		if got, err := FindParentMount("/non/existing/path"); err == nil {
			t.Errorf("expected an error for a non-existing path, got %q", got)
		}
	})

	// KNOWN ISSUE K4 (TEST_PLAN.md): mountpoints are matched as string prefix,
	// so the mount '<dir>/mnt' is taken for the parent of '<dir>/mntfoo'
	t.Run("sibling-of-mountpoint-with-same-prefix", func(t *testing.T) {
		if !IsRoot() {
			t.Skip("needs root permissions to create a bind mount")
		}
		dir, err := CleanPath(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		mnt := filepath.Join(dir, "mnt")
		sibling := filepath.Join(dir, "mntfoo")
		for _, d := range []string{mnt, sibling} {
			if err := os.Mkdir(d, 0755); err != nil {
				t.Fatal(err)
			}
		}
		if err := syscall.Mount(mnt, mnt, "", syscall.MS_BIND, ""); err != nil {
			t.Skipf("cannot create a bind mount: %v", err)
		}
		t.Cleanup(func() {
			if err := syscall.Unmount(mnt, syscall.MNT_DETACH); err != nil {
				t.Errorf("cannot unmount '%s': %v", mnt, err)
			}
		})

		if got, err := FindParentMount(filepath.Join(mnt, "below")); err == nil {
			t.Errorf("expected an error for a non-existing path, got %q", got)
		}
		want, err := FindParentMount(dir)
		if err != nil {
			t.Fatal(err)
		}
		got, err := FindParentMount(sibling)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("FindParentMount(%q) = %q, want %q", sibling, got, want)
		}
	})
}
