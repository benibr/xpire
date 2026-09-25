package helpers

import (
	"fmt"
	"github.com/moby/sys/mountinfo"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// return a clean absolute path
func CleanPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absPath, err = filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", err
	}
	absPath = filepath.Clean(absPath)
	return absPath, nil
}

// check if xpire runs as root
func IsRoot() bool {
	uid := os.Getuid()
	return uid == 0
}

// return the paths of the map sorted so that children come before
// their parents, to allow deleting nested subsets in one run.
// A child path always has its parent path as prefix,
// so descending string order is enough.
func SortPathsHierarchically(paths map[string]time.Time) []string {
	keys := make([]string, 0, len(paths))
	for p := range paths {
		keys = append(keys, p)
	}
	// this does only one sort round,
	// Reverse and StringSlice only prepare the data for the sort interface
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	return keys
}

// find the next upper path of mounted filesystem
func FindParentMount(path string) (string, error) {
	absPath, _ := CleanPath(path)
	//FIXME: logging does not work in the package
	//log.Debug(fmt.Sprintf("searching for parent mount of '%s'", absPath))
	// get all possible parents mounts
	mounts, _ := mountinfo.GetMounts(mountinfo.ParentsFilter(absPath))
	if len(mounts) == 0 {
		return "", fmt.Errorf("no parent mounts found for %s", absPath)
	}
	if len(mounts) > 1 {
		// usually we find at least two parents ;-)
		sort.Slice(mounts, func(i, j int) bool {
			return len(mounts[i].Mountpoint) > len(mounts[j].Mountpoint)
		})
	}
	//FIXME: logging does not work in the package
	//log.Debug(fmt.Sprintf("found parent mount '%s'", mounts[0].Mountpoint))
	return mounts[0].Mountpoint, nil
}
