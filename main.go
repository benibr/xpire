package main

import (
	"errors"
	"fmt"
	"os"
	"plugin"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/sirupsen/logrus"
)

const RC_OK = 0
const RC_ERR = 1
const RC_ERR_ARGS = 5
const RC_ERR_FS = 6
const RC_ERR_PLUGIN = 7

var log = logrus.New()

var args struct {
	SetExpireDate string `arg:"-s,--set"`
	Plugin        string `arg:"-p,--plugin"`
	Path          string
	Prune         bool
}

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

func getFsType(path string) (string, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return "", err
	}
	// from: https://github.com/torvalds/linux/blob/master/include/uapi/linux/magic.h
	// File system types in Linux (incomplete list)
	supportedFilesystems := map[int64]string{
		0x9123683E: "btrfs",
	}
	fsType, ok := supportedFilesystems[stat.Type]
	if !ok {
		return "", errors.New(fmt.Sprintf("unknown filesystem type: %x", stat.Type))
	}
	log.Info("detected filesystem: ", fsType)
	return fsType, nil
}

func loadPlugin(pluginName string) plugin.Plugin {
	pluginPath := fmt.Sprintf("./filesystems/%s.so", pluginName)
	plugin, err := plugin.Open(pluginPath)
	if err != nil {
		panic(err)
	}
	return *plugin
}

func main() {
	var pluginName string = ""
	var err error
	var parsedTime time.Time

	log = logrus.New()
	log.SetLevel(logrus.InfoLevel)
	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp:  true,
		FullTimestamp: false,
	})

	// args
	arg.MustParse(&args)
	checkPathArg()

	// plugin
	if args.Plugin == "" {
		pluginName, err = getFsType(args.Path)
		errorHandler(err, RC_ERR_FS, "Failed to detect filesystem type, use -p to specify a plugin")
	} else {
		pluginName = args.Plugin
	}
	plugin := loadPlugin(pluginName)
	initLoggerSym, err := plugin.Lookup("InitLogger")
	errorHandler(err, RC_ERR_PLUGIN, "Cannot find function 'InitLogger' in plugin")
	initLoggerFunc, ok := initLoggerSym.(func(*logrus.Logger) (error))
	okHandler(ok, RC_ERR_PLUGIN, "unexpected type from module symbol")
	err = initLoggerFunc(log)
	errorHandler(err, RC_ERR_PLUGIN, "Cannot initialize logger in plugin")

	// set expiration date
	if args.SetExpireDate != "" {
		if args.Prune {
			log.Error("Cannot use --prune with --setexpiredate")
			os.Exit(RC_ERR_ARGS)
		}
		parsedTime, err = time.Parse(time.DateTime, args.SetExpireDate)
		errorHandler(err, RC_ERR_ARGS, "Cannot parse specified date")
		setSym, err := plugin.Lookup("SetExpireDate")
		errorHandler(err, RC_ERR_PLUGIN, "Cannot find function 'SetExpireDate' in plugin")
		setFunc, ok := setSym.(func(time.Time, string) (error))
		okHandler(ok, RC_ERR_PLUGIN, "unexpected type from module symbol")
		log.Info(fmt.Sprintf("setting expiration date on snapshot '%s' to %s", args.Path, parsedTime.Format(time.DateTime)))
		err = setFunc(parsedTime, args.Path)
		errorHandler(err, RC_ERR_FS, "Error: Cannot set expiry date")
		os.Exit(RC_OK)

	// prune
	} else if args.Prune {
		pruneSym, err := plugin.Lookup("PruneExpiredSnapshots")
		errorHandler(err, RC_ERR_PLUGIN, "cann find function 'PruneExpiredSnapshots' in plugin")
		pruneFunc, ok := pruneSym.(func(string) ([]string, error))
		okHandler(ok, RC_ERR_PLUGIN, "unexpected type from module symbol")
		pruneFunc(args.Path)

	} else {
		log.Error("you have to specicy either --set-expire-date or --prune")
		os.Exit(RC_ERR_ARGS)
	}
}
