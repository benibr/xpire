package main

import (
	"time"

	"github.com/pkg/xattr"
)

// internal functions

// mandatory functions called by fsexpire
func SetExpireDate(t time.Time, path string) (bool, error) {
	//FIXME: check XATTR_SUPPORTED first
	//FIXME: we need to find the root btrfs subvolume first
	if err := xattr.SetWithFlags(path, "user.expire", []byte(t.Format(time.DateTime)), xattr.XATTR_REPLACE); err != nil {
		panic(err)
	}
	return true, nil
}

func PruneExpiredSnapshots(path string) ([]string, error) {
	return nil, nil
}

func main() {}
