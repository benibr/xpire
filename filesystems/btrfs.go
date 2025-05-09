package main

import (
	"fmt"
	"errors"
	"time"
	"path/filepath"
	"github.com/dennwc/btrfs"
	"github.com/pkg/xattr"
	"github.com/sirupsen/logrus"
)

const TimeFormat = time.DateTime

var log *logrus.Logger

// internal functions

// mandatory functions called by fsexpire

func InitLogger(l *logrus.Logger) error {
	log = l
	return nil
}

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
	log.Info(fmt.Sprintf("pruning all expired snapshots in '%s'", path))
	b, _ := btrfs.Open(path, false)
	subvols, _ := b.ListSubvolumes(func(svi btrfs.SubvolInfo) bool {
				if svi.RootID == 5 {
					fmt.Println("Refusing to work on btrfs <FS_TREE>")
					return false
				}
				return true
		})
	var expiredSubs []btrfs.SubvolInfo
	for _, sv := range subvols {
		fullPath := filepath.Join(path, sv.Path)
		xattr, err := xattr.Get(fullPath, "user.expire")
		if err != nil {
			log.Error(fmt.Sprintf("PruneExpiredSnapshots: %w", err))
			continue
		}
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
