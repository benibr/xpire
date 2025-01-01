package main

import (
	"fmt"
	"time"

	"github.com/dennwc/btrfs"
	"github.com/pkg/xattr"
)

// internal functions

// mandatory functions called by fsexpire
func SetExpireDate(t time.Time, path string) (bool, error) {
	//FIXME: check XATTR_SUPPORTED first
	//FIXME: we need to find the root btrfs subvolume first
	b, _ := btrfs.Open(path, false)
	fmt.Println(b.SubvolList())
	if err := xattr.Set(path, "user.expire", []byte(t.Format(time.DateTime))); err != nil {
		panic(err)
	}
	return true, nil
}

func PruneExpiredSnapshots(path string) ([]string, error) {
	return nil, nil
}

func main() {}
