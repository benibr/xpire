package main

import (
	"encoding/binary"
	"github.com/pkg/xattr"
	"time"
)

// internal functions

// mandatory functions called by fsexpire
func SetExpireDate(date time.Time, path string) (bool, error) {
	//FIXME: check XATTR_SUPPORTED first
	//FIXME: we need to find the root btrfs subvolume first
	unixTimestamp := date.Unix()
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(unixTimestamp))
	xattr.SetWithFlags(path, "user.expire", buf, xattr.XATTR_REPLACE)
	return true, nil
}

func PruneExpiredSnapshots(path string) ([]string, error) {
	return nil, nil
}

func main() {}
