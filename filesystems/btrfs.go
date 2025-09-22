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
	"os"
	"path/filepath"
	"time"
	"xpire/pluginapi"
)

const TimeFormat = time.DateTime
const RC_ERR_PLUGIN = 7

var (
	log *logrus.Logger
)

type BtrfsPlugin struct{}

// ---- internal functions

// check if xpire runs as root
func isRoot() bool {
	uid := os.Getuid()
	return uid == 0
}

// ---- mandatory functions called by fsexpire

func (p BtrfsPlugin) InitLogger(l *logrus.Logger) error {
	log = l
	return nil
}

func (p BtrfsPlugin) SetExpireDate(t time.Time, path string) error {
	//FIXME: check XATTR_SUPPORTED first
	isSubVolume, _ := btrfs.IsSubVolume(path)
	if isSubVolume == false {
		errorMsg := errors.New(fmt.Sprintf("'%s' is not a btrfs subvolume", path))
		return errorMsg
	}
	if err := xattr.Set(path, "user.expire", []byte(t.Format(TimeFormat))); err != nil {
		return fmt.Errorf("Failed to set xattr on '%s'\n%w", path, err)
	}
	return nil
}

func (p BtrfsPlugin) PruneExpired(path string) ([]string, error) {
	if !isRoot() {
		return nil, errors.New("btrfs plugin needs root permissions to list all subvolumes")
	}
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

// compile time check to verify that this plugin
// correctly implements the interface
var _ pluginapi.FsPluginApi = BtrfsPlugin{}

var FsPlugin = BtrfsPlugin{}
