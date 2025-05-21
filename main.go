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
	}
	fsType, ok := supportedFilesystems[stat.Type]
	if !ok {
		return "", errors.New(fmt.Sprintf("Filesystem not supported: %x.\nTry specifying the correct plugin explicitly with -p", stat.Type))
	}
	log.Info("Detected filesystem: ", fsType)
	return fsType, nil
}

// loadPlugin opens a filesystem plugin by name
func loadPlugin(pluginName string) plugin.Plugin {
	pluginPath := fmt.Sprintf("./filesystems/%s.so", pluginName)
	plugin, err := plugin.Open(pluginPath)
	if err != nil {
		panic(err)
	}
	return *plugin
}

func getPluginSymbol(p *plugin.Plugin, fName string) plugin.Symbol {
	log.Debug(fmt.Sprintf("Trying to find '%s' in plugin", fName))
	sym, err := p.Lookup(fName)
	errorHandler(err, RC_ERR_PLUGIN, fmt.Sprintf("Cannot find function '%s' in plugin", fName))
	return sym
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

	// plugin
	if args.Plugin == "" {
		pluginName, err = getFsType(args.Path)
		errorHandler(err, RC_ERR_FS, "Unable to autoselect plugin by detected filesystem:")
	} else {
		pluginName = args.Plugin
	}
	log.Debug(fmt.Sprintf("Loading plugin '%s'", pluginName))
	plugin := loadPlugin(pluginName)
	// init logging in plugin
	initLoggerSym := getPluginSymbol(&plugin, "InitLogger")
	initLoggerFunc, ok := initLoggerSym.(func(*logrus.Logger) error)
	okHandler(ok, RC_ERR_PLUGIN, "unexpected type from module symbol")
	initLoggerFunc(log)
	errorHandler(err, RC_ERR_PLUGIN, "Cannot initialize logger in plugin")

	// --set expiration date
	if args.SetExpireDate != "" {
		if args.Prune {
			log.Error("Cannot use --prune with --setexpiredate")
			os.Exit(RC_ERR_ARGS)
		}
		parsedTime, err = time.Parse(time.DateTime, args.SetExpireDate)
		errorHandler(err, RC_ERR_ARGS, "Cannot parse specified date")
		setSym := getPluginSymbol(&plugin, "SetExpireDate")
		setFunc, ok := setSym.(func(time.Time, string) error)
		okHandler(ok, RC_ERR_PLUGIN, "unexpected type from module symbol")
		log.Info(fmt.Sprintf("setting expiration date on snapshot '%s' to %s", args.Path, parsedTime.Format(time.DateTime)))
		err = setFunc(parsedTime, args.Path)
		errorHandler(err, RC_ERR_FS, "Error: Cannot set expiry date")

		// --prune expired data
	} else if args.Prune {
		pruneSym := getPluginSymbol(&plugin, "PruneExpiredSnapshots")
		pruneFunc, ok := pruneSym.(func(string) ([]string, error))
		okHandler(ok, RC_ERR_PLUGIN, "unexpected type from module symbol")
		pruneFunc(args.Path)

		// error in args
	} else {
		log.Error("you have to specicy either --set or --prune")
		os.Exit(RC_ERR_ARGS)
	}
	os.Exit(RC_OK)
}
