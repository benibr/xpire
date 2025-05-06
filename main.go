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
const RC_ERR_ARGS = 5
const RC_ERR_FS = 6

var log = logrus.New()

var args struct {
	SetExpireDate string `arg:"-s,--set-expire-date"`
	Plugin        string `arg:"-p,--plugin"`
	Path          string
	Prune         bool
}

func checkPath() {
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
	})

	arg.MustParse(&args)

	checkPath()
	if args.Plugin == "" {
		pluginName, err = getFsType(args.Path)
		if err != nil {
			log.Error(err)
			os.Exit(RC_ERR_FS)
		}
	} else {
		pluginName = args.Plugin
	}

	// set expiration date
	if args.SetExpireDate != "" {
		if args.Prune {
			log.Error("Cannot use --prune with --setexpiredate")
			os.Exit(RC_ERR_ARGS)
		}
		parsedTime, err = time.Parse(time.DateTime, args.SetExpireDate)
		if err != nil {
			log.Error(err)
			os.Exit(RC_ERR_ARGS)
		}
		plugin := loadPlugin(pluginName)
		setSym, err := plugin.Lookup("SetExpireDate")
		if err != nil {
			panic(err)
		}
		setFunc, ok := setSym.(func(time.Time, string) (error))
		if !ok {
			panic("unexpected type from module symbol")
		}
		log.Info("setting expiration date on snapshot '%s' to %s\n", args.Path, parsedTime.Format(time.DateTime))
		err = setFunc(parsedTime, args.Path)
		if err != nil {
			log.Error("Error: Cannot set expiry date")
			log.Error(err)
			os.Exit(RC_ERR_FS)
		}
		os.Exit(RC_OK)

	// prune
	} else if args.Prune {
		plugin := loadPlugin(pluginName)
		pruneSym, err := plugin.Lookup("PruneExpiredSnapshots")
		if err != nil {
			panic(err)
		}
		pruneFunc, ok := pruneSym.(func(string) ([]string, error))
		if !ok {
			log.Error(ok)
			panic("unexpected type from module symbol")
		}
		pruneFunc(args.Path)

	} else {
		log.Error("you have to specicy either --set-expire-date or --prune")
	}
}
