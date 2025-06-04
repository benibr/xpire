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
	"github.com/dennwc/btrfs"
	"github.com/pkg/xattr"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"time"
)

const TimeFormat = time.DateTime

var log *logrus.Logger

// internal functions

// mandatory functions called by fsexpire

func InitLogger(l *logrus.Logger) error {
	log = l
	return nil
}

func SetExpireDate(t time.Time, path string) error {
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

func PruneExpired(path string) ([]string, error) {
	log.Info(fmt.Sprintf("pruning expired data in '%s'", path))
	b, _ := btrfs.Open(path, false)
	subvols, _ := b.ListSubvolumes(func(svi btrfs.SubvolInfo) bool {
		if svi.RootID == 5 {
			log.Debug("Refusing to work on btrfs <FS_TREE>")
			return false
		}
		return true
	})
	var expiredSubs []btrfs.SubvolInfo
	for _, sv := range subvols {
		fullPath := filepath.Join(path, sv.Path)
		xattr, err := xattr.Get(fullPath, "user.expire")
		if err != nil {
			log.Debug(fmt.Sprintf("Function PruneExpired: %w", err))
			continue
		}
		t, err := time.Parse(TimeFormat, string(xattr))
		if err != nil {
			panic(err)
		}
		if t.Before(time.Now()) {
			expiredSubs = append(expiredSubs, sv)
			log.Info(fmt.Sprintf("↳ '%s' expired since %s", fullPath, t.Format(TimeFormat)))
			btrfs.DeleteSubVolume(fullPath)
		}
	}
	return nil, nil
}

func main() {}
