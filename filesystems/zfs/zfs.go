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
	"fmt"
	// zfs "github.com/bicomsystems/go-libzfs" //outdated
	zfs "github.com/bitomia/go-libzfs"
	"github.com/pkg/xattr"
	"github.com/sirupsen/logrus"
	"os"
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

type ZfsPlugin struct{}

// ---- internal functions

// check if xpire runs as root
func isRoot() bool {
	uid := os.Getuid()
	return uid == 0
}

// ---- mandatory functions called by fsexpire

func (p ZfsPlugin) InitLogger(l *logrus.Logger) error {
	log = l
	return nil
}

func (p ZfsPlugin) SetExpireDate(t time.Time, path string) error {
	absPath, err := helpers.CleanPath(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// this plugin only works on datasets so we need to check
	// if absPath is a valid, mounted ZFS dataset
	dataset, err := zfs.DatasetOpen(absPath)
	if err != nil {
		return fmt.Errorf("failed to list ZFS datasets: %w", err)
	}
	defer dataset.Close()
	var isDataset bool
	ds := dataset
	//for _, ds := range datasets {
	isMounted, mountpoint := ds.IsMounted()
	log.Debug(fmt.Sprintf("DEBUG: %s", isMounted))
	log.Debug(fmt.Sprintf("DEBUG: %s", mountpoint))
	log.Debug(fmt.Sprintf("DEBUG: %s", absPath))
	pp := ds.PoolName()
	log.Debug(fmt.Sprintf("DEBUG: %s", pp))
	if (isMounted == true) && (mountpoint == absPath) {
		log.Debug(fmt.Sprintf("WEhat if"))
		isDataset = true
		//break
	}
	//}
	if !isDataset {
		return fmt.Errorf("'%s' is not a valid, mounted ZFS dataset", absPath)
	}
	if err := xattr.Set(path, "user.expire", []byte(t.Format(TimeFormat))); err != nil {
		return fmt.Errorf("Failed to set xattr on '%s'\n%w", path, err)
	}
	return nil
}

func (p ZfsPlugin) PruneExpired(path string) ([]string, error) {
	return nil, nil
}

func main() {}

// compile time check to verify that this plugin
// correctly implements the interface
var _ pluginapi.FsPluginApi = ZfsPlugin{}

var FsPlugin = ZfsPlugin{}
