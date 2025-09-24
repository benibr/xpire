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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
func cleanPath(path string) (string, error) {
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

func findParentMount(path string) (string, error) {
	absPath, _ := cleanPath(path)
	log.Debug(fmt.Sprintf("searching for parent mount of '%s'", absPath))
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
	log.Debug(fmt.Sprintf("found parent mount '%s'", mounts[0].Mountpoint))
	return mounts[0].Mountpoint, nil
}

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
		errorMsg := fmt.Errorf("'%s' is not a btrfs subvolume", path)
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

	// next parent mountpoint of path is the btrfs filesystem we work on
	absPath, _ := cleanPath(path)
	var mountPoint string = absPath
	pathIsMountpoint, err := mountinfo.Mounted(absPath)
	if pathIsMountpoint == false {
		mountPoint, _ = findParentMount(absPath)
	}
	b, err := btrfs.Open(mountPoint, false)
	if err != nil {
		return nil, fmt.Errorf("Cannot open btrfs filesystem\n%w", err)
	}

	// remove mountpoint path from given path to guess the subvolume names
	var relPath string = strings.Replace(absPath, mountPoint, "", 1)
	relPath = strings.TrimLeft(relPath, "/")
	pathIsSubvolume, _ := btrfs.IsSubVolume(absPath)
	if pathIsSubvolume == true {
		pathVolume, _ := b.SubvolumeByPath(absPath)
		log.Debug(fmt.Sprintf("Subvolume ID of '%s' is '%d'", absPath, pathVolume.RootID))
	}
	subvols, _ := b.ListSubvolumes(func(svi btrfs.SubvolInfo) bool {
		if svi.RootID == 5 {
			log.Debug("Refusing to work on btrfs <FS_TREE>")
			return false
		}
		if strings.HasPrefix(svi.Path, relPath) {
			return true
		}
		return false
	})
	for _, sv := range subvols {
		// FIXME: BUG! This is wrong if the path is not the mountpoint of btrfs FS
		// when path is a subvolume, then we need to filter the all subvolumes of b and check if they start with name of the given subvolume
		fullPath := filepath.Join(mountPoint, sv.Path)
		log.Debug(fmt.Sprintf("Working on path '%s'", fullPath))
		xattr, err := xattr.Get(fullPath, "user.expire")
		if err != nil {
			log.Debug(fmt.Errorf("Cannot read expire xattr on '%s'\n\t%w", fullPath, err))
			continue
		}
		t, err := time.Parse(TimeFormat, string(xattr))
		if err != nil {
			log.Warn(fmt.Errorf("Cannot parse expire date format:\n\t%w", err))
			continue
		}
		if t.Before(time.Now()) {
			log.Info(fmt.Sprintf("↳ Subvolume '%s' expired since %s", sv.Path, t.Format(TimeFormat)))
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
