package main

import (
	"fmt"
	"errors"
	"time"
	"path/filepath"
	"github.com/dennwc/btrfs"
	"github.com/pkg/xattr"
)

const TimeFormat = time.DateTime
// internal functions

// mandatory functions called by fsexpire
func SetExpireDate(t time.Time, path string) (error) {
	//FIXME: check XATTR_SUPPORTED first
	isSubVolume, _ := btrfs.IsSubVolume(path)
	if isSubVolume == false {
		errorMsg := errors.New(fmt.Sprintf("'%s' is not a btrfs subvolume", path))
		return errorMsg
	}
	if err := xattr.Set(path, "user.expire", []byte(t.Format(TimeFormat))); err != nil {
		panic(err)
	}
	return nil
}

func PruneExpiredSnapshots(path string) ([]string, error) {
	fmt.Printf("pruning all expired snapshots in '%s'\n", path)
	b, _ := btrfs.Open(path, false)
	subvols, _ := b.ListSubvolumes(func(svi btrfs.SubvolInfo) bool {
        return true // no filter, return all subvolumes
    })
  var expiredSubs []btrfs.SubvolInfo
	for _, sv := range subvols {
		fullPath := filepath.Join(path, sv.Path)
		xattr, err := xattr.Get(fullPath, "user.expire")
		if err != nil { continue }
		t, err := time.Parse(TimeFormat, string(xattr))
		if err != nil {
			panic(err)
		}
		if t.Before(time.Now()) {
			expiredSubs = append(expiredSubs, sv)
			fmt.Print("  '")
			fmt.Print(fullPath)
			fmt.Print("' expired since ")
			fmt.Println(t.Format(TimeFormat))
			btrfs.DeleteSubVolume(fullPath)
		}
	}
	return nil, nil
}

func main() {}
