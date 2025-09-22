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
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/sirupsen/logrus"
	"xpire/pluginapi"
)

// global const
const RC_OK = 0
const RC_ERR = 1
const RC_ERR_ARGS = 5
const RC_ERR_FS = 6
const RC_ERR_PLUGIN = 7

// global vars
var log = logrus.New()

// functions
var args struct {
	SetExpireDate string `arg:"-s,--set"`
	Plugin        string `arg:"-p,--plugin"`
	Path          string
	Prune         bool
	Loglevel      string `arg:"-l,--loglevel"`
}

func main() {
	// vars
	var pluginName string = ""
	var err error
	var parsedTime time.Time

	// logging
	log = logrus.New()
	log.SetLevel(logrus.InfoLevel)
	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
		FullTimestamp:    false,
	})

	// args
	arg.MustParse(&args)
	checkPathArg()
	if args.Loglevel != "" {
		l, err := logrus.ParseLevel(args.Loglevel)
		errorHandler(err, RC_ERR_ARGS, "Unknown Loglevel, see https://pkg.go.dev/github.com/sirupsen/logrus#Level")
		log.SetLevel(l)
	}

	// plugin
	if args.Plugin == "" {
		pluginName, err = getFsType(args.Path)
		errorHandler(err, RC_ERR_FS, "Unable to autoselect plugin by detected filesystem")
	} else {
		pluginName = args.Plugin
	}
	log.Debug(fmt.Sprintf("Loading plugin '%s'", pluginName))
	p, err := loadPlugin(pluginName)
	errorHandler(err, RC_ERR_PLUGIN, fmt.Sprintf("Cannot load plugin '%s'", pluginName))
	symPlugin, err := p.Lookup("FsPlugin")
	errorHandler(err, RC_ERR_PLUGIN, fmt.Sprintf("Cannot lookup plugin type 'FsPlugin' in plugin '%s'", pluginName))
	fsplugin, ok := symPlugin.(pluginapi.FsPluginApi)
	okHandler(ok, RC_ERR_PLUGIN, "Unexpected type from plugin symbol")
	err = fsplugin.InitLogger(log)
	errorHandler(err, RC_ERR_PLUGIN, "Cannot initialize logger in plugin")

	// --set expiration date
	if args.SetExpireDate != "" {
		if args.Prune {
			log.Error("Cannot use --prune with --set")
			os.Exit(RC_ERR_ARGS)
		}
		parsedTime, err = time.Parse(time.DateTime, args.SetExpireDate)
		errorHandler(err, RC_ERR_ARGS, "Cannot parse specified date")
		log.Info(fmt.Sprintf("setting expiration date on '%s' to %s", args.Path, parsedTime.Format(time.DateTime)))
		err = fsplugin.SetExpireDate(parsedTime, args.Path)
		errorHandler(err, RC_ERR_FS, "Error: Cannot set expiry date")

		// --prune expired data
	} else if args.Prune {
		_, err = fsplugin.PruneExpired(args.Path)
		errorHandler(err, RC_ERR_PLUGIN, fmt.Sprintf("Error during pruning:\n %s", err))

		// error in args
	} else {
		log.Error("you have to specicy either --set or --prune")
		os.Exit(RC_ERR_ARGS)
	}
	os.Exit(RC_OK)
}
