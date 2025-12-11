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
	"github.com/moby/sys/mountinfo"
	"github.com/pkg/xattr"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strings"
	"time"
	"xpire/helpers"
	"xpire/pluginapi"
)

const TimeFormat = time.DateTime
const RC_OK = 0
const RC_ERR_PLUGIN = 7

var (
	log *logrus.Logger
)

type BtrfsPlugin struct{}

// ---- internal functions
func findParentBtrfs(path string) (string, error) {
	var mountPoint = path
	pathIsMountpoint, _ := mountinfo.Mounted(path)
	if !pathIsMountpoint {
		mountPoint, _ = helpers.FindParentMount(path)
	}
	return mountPoint, nil
}

func findChildSubvolumes(absPath string, mountPoint string, b *btrfs.FS) []btrfs.SubvolInfo {
	// remove mountpoint path from given path to guess the subvolume names
	var relPath = strings.Replace(absPath, mountPoint, "", 1)
	relPath = strings.TrimLeft(relPath, "/")

	subvols, _ := b.ListSubvolumes(func(svi btrfs.SubvolInfo) bool {
		if svi.RootID == 5 {
			log.Debug("Refusing to work on btrfs <FS_TREE>")
			return false
		}
		// return only subvolumes that have the prefix of the given path
		if strings.HasPrefix(svi.Path, relPath) {
			return true
		}
		return false
	})
	return subvols
}

// ---- mandatory functions called by fsexpire
func (p BtrfsPlugin) InitLogger(l *logrus.Logger) error {
	log = l
	return nil
}

func (p BtrfsPlugin) UnsetExpireDate(path string) error {
	return nil
}

func (p BtrfsPlugin) SetExpireDate(t time.Time, path string) error {
	//FIXME: check XATTR_SUPPORTED first
	isSubVolume, _ := btrfs.IsSubVolume(path)
	if !isSubVolume {
		errorMsg := fmt.Errorf("'%s' is not a btrfs subvolume", path)
		return errorMsg
	}
	if err := xattr.Set(path, "user.expire", []byte(t.Format(TimeFormat))); err != nil {
		return fmt.Errorf("failed to set xattr on '%s'\n%w", path, err)
	}
	return nil
}

func (p BtrfsPlugin) PruneExpired(path string) ([]string, error) {
	if !helpers.IsRoot() {
		return nil, errors.New("btrfs plugin needs root permissions to list all subvolumes")
	}
	log.Info(fmt.Sprintf("pruning expired data in '%s'", path))

	absPath, _ := helpers.CleanPath(path)

	// next parent mountpoint of path is the btrfs filesystem we work on
	mountPoint, err := findParentBtrfs(absPath)

	b, err := btrfs.Open(mountPoint, false)
	if err != nil {
		return nil, fmt.Errorf("cannot open btrfs filesystem\n%w", err)
	}

	subvols := findChildSubvolumes(absPath, mountPoint, b)

	// iterate over all subvolumes and delete them if their expire date is reached
	for _, sv := range subvols {
		fullPath := filepath.Join(mountPoint, sv.Path)
		log.Debug(fmt.Sprintf("Working on path '%s'", fullPath))
		xattr, err := xattr.Get(fullPath, "user.expire")
		if err != nil {
			log.Debug(fmt.Errorf("cannot read expire xattr on '%s'\n\t%w", fullPath, err))
			continue
		}
		t, err := time.Parse(TimeFormat, string(xattr))
		if err != nil {
			log.Warn(fmt.Errorf("cannot parse expire date format:\n\t%w", err))
			continue
		}
		if t.Before(time.Now()) {
			log.Info(fmt.Sprintf("↳ Subvolume '%s' expired since %s", sv.Path, t.Format(TimeFormat)))
			if err := btrfs.DeleteSubVolume(fullPath); err != nil {
				// Handle the error appropriately, e.g., log or return
				log.Printf("failed to delete subvolume %s: %v", fullPath, err)
			}
		}
	}
	// TODO: return list of deleted paths not yet implemented
	return nil, nil
}

func (p BtrfsPlugin) List(path string) ([]string, error) {
	if !helpers.IsRoot() {
		return nil, errors.New("btrfs plugin needs root permissions to list all subvolumes")
	}
	absPath, _ := helpers.CleanPath(path)
	log.Info(fmt.Sprintf("searching for all expire dates in '%s'", absPath))

	// next parent mountpoint of path is the btrfs filesystem we work on
	mountPoint, err := findParentBtrfs(absPath)

	b, err := btrfs.Open(mountPoint, false)
	if err != nil {
		return nil, fmt.Errorf("cannot open btrfs filesystem\n%w", err)
	}

	subvols := findChildSubvolumes(absPath, mountPoint, b)

	// iterate over all subvolumes and show their expiration date
	for _, sv := range subvols {
		//FIXME: isn't this the same as absPath?
		fullPath := filepath.Join(mountPoint, sv.Path)
		log.Debug(fmt.Sprintf("Working on path '%s'", fullPath))
		xattr, err := xattr.Get(fullPath, "user.expire")
		if err != nil {
			log.Debug(fmt.Errorf("cannot read expire xattr on '%s'\n\t%w", fullPath, err))
			continue
		}
		t, err := time.Parse(TimeFormat, string(xattr))
		if err != nil {
			log.Warn(fmt.Errorf("cannot parse expire date format:\n\t%w", err))
			continue
		}
		if t.Before(time.Now()) {
			log.Info(fmt.Sprintf("↳ Subvolume '%s' expired since %s", sv.Path, t.Format(TimeFormat)))
		} else {
			log.Info(fmt.Sprintf("↳ Subvolume '%s' expires in %s", sv.Path, t.Format(TimeFormat)))
		}
	}
	// TODO: return list of paths not yet implemented
	return nil, nil
}

func main() {}

// compile time check to verify that this plugin
// correctly implements the interface
var _ pluginapi.FsPluginApi = BtrfsPlugin{}

var FsPlugin = BtrfsPlugin{}
