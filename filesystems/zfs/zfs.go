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
	"os/exec"
	"strings"
	"time"
	"xpire/helpers"
	"xpire/pluginapi"
	// zfs "github.com/bicomsystems/go-libzfs" //outdated
	//zfs "github.com/bitomia/go-libzfs"
	zfs "github.com/mistifyio/go-zfs"
	"github.com/pkg/xattr"
	"github.com/sirupsen/logrus"
)

const TimeFormat = time.DateTime
const RC_OK = 0
const RC_ERR_PLUGIN = 7

var (
	log *logrus.Logger
)

type ZfsPlugin struct{}

// ---- internal functions

// return the value of a ZFS dataset property
// because non of the tested golang ZFS libraries got the property values correctly
// so we do this manually by parsing the ZFS command output
func zfsGet(dataset, property string) (string, error) {
	cmd := exec.Command("zfs", "get", "-H", property, dataset)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return parseZfsGetOutput(string(output)), nil
}

// return the value column of the output of 'zfs get -H'
func parseZfsGetOutput(output string) string {
	fields := strings.Split(strings.TrimSpace(output), "\t")
	if len(fields) < 3 {
		return ""
	}

	return fields[2]
}

// check if a dataset mountpoint lies under the given path
func mountpointUnder(mountpoint, absPath string) bool {
	if mountpoint == absPath {
		return true
	}
	return strings.HasPrefix(mountpoint, strings.TrimSuffix(absPath, "/")+"/")
}

// ---- mandatory functions called by fsexpire

func (p ZfsPlugin) InitLogger(l *logrus.Logger) error {
	log = l
	return nil
}

func (p ZfsPlugin) UnsetExpireDate(path string) error {
	absPath, err := helpers.CleanPath(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}
	// this plugin only works on datasets so we need to check
	// if absPath is a valid, mounted ZFS dataset
	datasets, err := zfs.Datasets("")
	if err != nil {
		return fmt.Errorf("failed to list ZFS datasets: %w", err)
	}
	var isDataset bool
	// for unknown reasons ds.IsMounted returns the mountpoint of the FS not the DS
	// so we iterate through all DSs and check if one return
	for _, ds := range datasets {
		isMounted, _ := zfsGet(ds.Name, "mounted")
		if isMounted == "yes" {
			mountpoint, _ := zfsGet(ds.Name, "mountpoint")
			if mountpoint == absPath {
				isDataset = true
				break
			}
		} else {
			continue
		}
	}
	if !isDataset {
		return fmt.Errorf("'%s' is not a valid, mounted ZFS dataset", absPath)
	}
	if err := xattr.Remove(path, "user.expire"); err != nil {
		return fmt.Errorf("failed to remove xattr on '%s'\n%w", path, err)
	}
	return nil
}

func (p ZfsPlugin) SetExpireDate(t time.Time, path string) error {
	absPath, err := helpers.CleanPath(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}
	// this plugin only works on datasets so we need to check
	// if absPath is a valid, mounted ZFS dataset
	datasets, err := zfs.Datasets("")
	if err != nil {
		return fmt.Errorf("failed to list ZFS datasets: %w", err)
	}
	var isDataset bool
	// for unknown reasons ds.IsMounted returns the mountpoint of the FS not the DS
	// so we iterate through all DSs and check if one return
	for _, ds := range datasets {
		isMounted, _ := zfsGet(ds.Name, "mounted")
		if isMounted == "yes" {
			mountpoint, _ := zfsGet(ds.Name, "mountpoint")
			if mountpoint == absPath {
				isDataset = true
				break
			}
		} else {
			continue
		}
	}
	if !isDataset {
		return fmt.Errorf("'%s' is not a valid, mounted ZFS dataset", absPath)
	}
	if err := xattr.Set(path, "user.expire", []byte(t.Format(TimeFormat))); err != nil {
		return fmt.Errorf("failed to set xattr on '%s'\n%w", path, err)
	}
	return nil
}

func (p ZfsPlugin) PruneExpired(path string) ([]string, error) {
	absPath, _ := helpers.CleanPath(path)
	log.Info(fmt.Sprintf("pruning expired data in '%s'", path))

	datasets, err := zfs.Datasets("")
	if err != nil {
		return nil, fmt.Errorf("failed to list ZFS datasets: %w", err)
	}
	// continue with other datasets if one cannot be destroyed
	var destroyErrs []error
	for _, ds := range datasets {
		mountpoint, _ := zfsGet(ds.Name, "mountpoint")
		if !mountpointUnder(mountpoint, absPath) {
			continue
		}
		log.Debug(fmt.Sprintf("Checking path '%s'", mountpoint))
		isMounted, _ := zfsGet(ds.Name, "mounted")
		if isMounted == "yes" {
			xattr, err := xattr.Get(mountpoint, "user.expire")
			if err != nil {
				log.Debug(fmt.Errorf("cannot read expire xattr on '%s'\n\t%w", mountpoint, err))
				continue
			}
			t, err := time.Parse(TimeFormat, string(xattr))
			if err != nil {
				log.Warn(fmt.Errorf("cannot parse expire date format:\n\t%w", err))
				continue
			}
			if t.Before(time.Now()) {
				log.Info(fmt.Sprintf("↳ Dataset '%s' expired since %s", ds.Name, t.Format(TimeFormat)))
				if err := ds.Destroy(0); err != nil {
					destroyErrs = append(destroyErrs, fmt.Errorf("failed to destroy dataset '%s'\n%w", ds.Name, err))
				}
			}
		} else {
			log.Debug(fmt.Sprintf("skipping unmounted path '%s'", mountpoint))
			continue
		}
	}
	return nil, errors.Join(destroyErrs...)
}

func (ZfsPlugin) List(path string) ([]string, error) {
	absPath, _ := helpers.CleanPath(path)
	log.Info(fmt.Sprintf("Listing data in '%s'", path))

	datasets, err := zfs.Datasets("")
	if err != nil {
		return nil, fmt.Errorf("failed to list ZFS datasets: %w", err)
	}
	for _, ds := range datasets {
		mountpoint, _ := zfsGet(ds.Name, "mountpoint")
		if !mountpointUnder(mountpoint, absPath) {
			continue
		}
		log.Debug(fmt.Sprintf("Checking path '%s'", mountpoint))
		isMounted, _ := zfsGet(ds.Name, "mounted")
		if isMounted == "yes" {
			xattr, err := xattr.Get(mountpoint, "user.expire")
			if err != nil {
				log.Debug(fmt.Errorf("cannot read expire xattr on '%s'\n\t%w", mountpoint, err))
				continue
			}
			t, err := time.Parse(TimeFormat, string(xattr))
			if err != nil {
				log.Warn(fmt.Errorf("cannot parse expire date format:\n\t%w", err))
				continue
			}
			if t.Before(time.Now()) {
				log.Info(fmt.Sprintf("↳ Dataset '%s' expired since %s", ds.Name, t.Format(TimeFormat)))
			} else {
				log.Info(fmt.Sprintf("↳ Dataset '%s' expires in %s", ds.Name, t.Format(TimeFormat)))
			}
		} else {
			log.Debug(fmt.Sprintf("skipping unmounted path '%s'", mountpoint))
			continue
		}
	}
	return nil, nil
}

func main() {}

// compile time check to verify that this plugin
// correctly implements the interface
var _ pluginapi.FsPluginApi = ZfsPlugin{}

var FsPlugin = ZfsPlugin{}
