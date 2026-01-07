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
	"path"
	"path/filepath"
	"strings"
	"time"
	"xpire/helpers"
	"xpire/pluginapi"

	"github.com/dennwc/btrfs"
	"github.com/moby/sys/mountinfo"
	"github.com/pkg/xattr"
	"github.com/sirupsen/logrus"
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

// findMountRoot returns the "mount root": the path of the subvolume that is
// mounted at mountPoint, relative to the top level subvolume of the btrfs
// filesystem. A btrfs filesystem can be mounted as a whole or with
// '-o subvol=<path>', in which case only that subvolume is visible below
// mountPoint:
//
//	mount /dev/sdx /mnt                        -> ""
//	mount -o subvol=subvolume01 /dev/sdx /mnt  -> "subvolume01"
//
// btrfs itself always names subvolumes relative to the top level subvolume
// (e.g. in ListSubvolumes), so the mount root is the offset needed to
// translate between those names and paths in the mounted directory tree,
// see subvolumeRelPath and subvolumeFullPath.
func findMountRoot(mountPoint string) (string, error) {
	mounts, err := mountinfo.GetMounts(mountinfo.SingleEntryFilter(mountPoint))
	if err != nil {
		return "", err
	}
	if len(mounts) == 0 {
		return "", fmt.Errorf("no mount found at '%s'", mountPoint)
	}
	return strings.Trim(mounts[len(mounts)-1].Root, "/"), nil
}

// subvolumeRelPath translates a path in the mounted directory tree into the
// subvolume path as btrfs names it: it strips the mountPoint and prepends
// the mountRoot. It is the inverse of subvolumeFullPath.
// Example with mountPoint '/mnt' and mountRoot 'subvolume01':
//
//	'/mnt/child' -> 'subvolume01/child'
//	'/mnt'       -> 'subvolume01'
func subvolumeRelPath(absPath string, mountPoint string, mountRoot string) string {
	// remove mountpoint path from given path to guess the subvolume names
	var relPath = strings.Replace(absPath, mountPoint, "", 1)
	relPath = strings.TrimLeft(relPath, "/")
	return strings.TrimLeft(path.Join(mountRoot, relPath), "/")
}

// subvolumeFullPath translates a subvolume path as btrfs names it into the
// path in the mounted directory tree: it strips the mountRoot and prepends
// the mountPoint. It is the inverse of subvolumeRelPath.
// Example with mountPoint '/mnt' and mountRoot 'subvolume01':
//
//	'subvolume01/child' -> '/mnt/child'
//	'subvolume01'       -> '/mnt'
func subvolumeFullPath(mountPoint string, mountRoot string, svPath string) string {
	return filepath.Join(mountPoint, strings.TrimPrefix(svPath, mountRoot))
}

func findChildSubvolumes(absPath string, mountPoint string, mountRoot string, b *btrfs.FS) []btrfs.SubvolInfo {
	relPath := subvolumeRelPath(absPath, mountPoint, mountRoot)

	subvols, _ := b.ListSubvolumes(func(svi btrfs.SubvolInfo) bool {
		if svi.RootID == 5 {
			log.Debug("Refusing to work on btrfs <FS_TREE>")
			return false
		}
		return subvolumeUnder(svi.Path, relPath)
	})
	return subvols
}

// return only subvolumes that have the prefix of the given path
func subvolumeUnder(svPath string, relPath string) bool {
	if relPath == "" || svPath == relPath {
		return true
	}
	return strings.HasPrefix(svPath, relPath+"/")
}

// ---- mandatory functions called by fsexpire
func (p BtrfsPlugin) InitLogger(l *logrus.Logger) error {
	log = l
	return nil
}

func (p BtrfsPlugin) UnsetExpireDate(path string) error {
	//FIXME: check XATTR_SUPPORTED first
	isSubVolume, _ := btrfs.IsSubVolume(path)
	if !isSubVolume {
		errorMsg := fmt.Errorf("'%s' is not a btrfs subvolume", path)
		return errorMsg
	}
	if err := xattr.Remove(path, "user.expire"); err != nil {
		return fmt.Errorf("failed to remove xattr on '%s'\n%w", path, err)
	}
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

func (p BtrfsPlugin) PruneExpired(paths map[string]time.Time) error {
	log.Info("pruning expired data")
	var deleteErrs []error
	for path, date := range paths {
		log.Debug(fmt.Sprintf("checking path='%s', date='%s'", path, date.Format(TimeFormat)))
		absPath, _ := helpers.CleanPath(path)
		if date.Before(time.Now()) {
			log.Info(fmt.Sprintf("↳ '%s' expired since %s", absPath, date.Format(TimeFormat)))
			if err := btrfs.DeleteSubVolume(absPath); err != nil {
				log.Printf("failed to delete subvolume %s: %v", absPath, err)
				deleteErrs = append(deleteErrs, fmt.Errorf("failed to delete subvolume '%s'\n%w", absPath, err))
			}
		}
	}
	return errors.Join(deleteErrs...)
}

func (p BtrfsPlugin) List(path string) (map[string]time.Time, error) {
	ret := make(map[string]time.Time)

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

	mountRoot, err := findMountRoot(mountPoint)
	if err != nil {
		return nil, fmt.Errorf("cannot find mounted btrfs subvolume\n%w", err)
	}

	subvols := findChildSubvolumes(absPath, mountPoint, mountRoot, b)

	// iterate over all subvolumes and show their expiration date
	for _, sv := range subvols {
		//FIXME: isn't this the same as absPath?
		fullPath := subvolumeFullPath(mountPoint, mountRoot, sv.Path)
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
		ret[fullPath] = t
	}
	return ret, nil
}

func main() {}

// compile time check to verify that this plugin
// correctly implements the interface
var _ pluginapi.FsPluginApi = BtrfsPlugin{}

var FsPlugin = BtrfsPlugin{}
