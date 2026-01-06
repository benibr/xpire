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
	"os"
	"plugin"
	"syscall"
)

func errorHandler(err error, rc int, msg string) {
	if err != nil {
		log.Error(err)
		os.Exit(rc)
	}
}

func okHandler(ok bool, rc int, msg string) {
	if !ok {
		log.Error(msg)
		os.Exit(rc)
	}
}

func checkPathArg() {
	if args.Path == "" {
		log.Error("--path missing")
		os.Exit(RC_ERR_ARGS)
	}
}

// getFsType detects the filesystem of a given path
// via a list of supported ones
func getFsType(path string) (string, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return "", err
	}
	// from: https://github.com/torvalds/linux/blob/master/include/uapi/linux/magic.h
	supportedFilesystems := map[int64]string{
		0x9123683E: "btrfs",
		0x2FC12FC1: "zfs",
	}
	fsType, ok := supportedFilesystems[stat.Type]
	if !ok {
		return "", errors.New(fmt.Sprintf("Filesystem not supported: %x.\nTry specifying the correct plugin explicitly with -p", stat.Type))
	}
	log.Debug("Detected filesystem: ", fsType)
	return fsType, nil
}

// loadPlugin opens a filesystem plugin by name
func loadPlugin(pluginName string) (plugin.Plugin, error) {
	pluginPath := fmt.Sprintf("./filesystems/%s/%s.so", pluginName, pluginName)
	plugin, err := plugin.Open(pluginPath)
	return *plugin, err
}
