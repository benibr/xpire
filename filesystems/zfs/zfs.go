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
