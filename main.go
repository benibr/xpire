package main

import (
	"errors"
	"fmt"
	"github.com/alexflint/go-arg"
	"os"
	"plugin"
	"syscall"
	"time"
)

const RC_OK = 0
const RC_ERR_ARGS = 5
const RC_ERR_FS = 6

var args struct {
	SetExpireDate string `arg:"-s,--set-expire-date"`
	Path          string
	Prune         bool
}

func printError(msg string) {
	fmt.Println("Error:", msg)
}

func checkPath() {
	if args.Path == "" {
		printError("--path missing")
		os.Exit(RC_ERR_ARGS)
	}
}

func getFsType(path string) (string, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return "", err
	}

	// File system types in Linux (incomplete list)
	supportedFilesystems := map[int64]string{
		0x9123683E: "btrfs",
	}

	fsType, ok := supportedFilesystems[stat.Type]
	if !ok {
		return "", errors.New(fmt.Sprintf("unknown filesystem type: %x", stat.Type))
	}
	return fsType, nil
}

func main() {

	arg.MustParse(&args)

	// set expiration date
	if args.SetExpireDate != "" {
		checkPath()
		if args.Prune {
			printError("Cannot use --prune with --setexpiredate")
			os.Exit(RC_ERR_ARGS)
		}
		parsedTime, err := time.Parse(time.DateTime, args.SetExpireDate)
		if err != nil {
			fmt.Println(err)
			os.Exit(RC_ERR_ARGS)
		}
		fmt.Printf("setting expiration date on snapshot '%s' to %s\n", args.Path, parsedTime.Format(time.DateTime))
		os.Exit(RC_OK)

	} else if args.Prune {
		checkPath()
		fmt.Printf("pruning all expired snapshots in '%s'\n", args.Path)
		fsType, err := getFsType(args.Path)
		if err != nil {
			fmt.Println(err)
			os.Exit(RC_ERR_FS)
		}
		fmt.Println("detected filesystem:", fsType)
		pluginPath := fmt.Sprintf("./filesystems/%s.so", fsType)
		p, err := plugin.Open(pluginPath)
		if err != nil {
			panic(err)
		}
		pruneSym, err := p.Lookup("PruneExpiredSnapshots")
		if err != nil {
			panic(err)
		}
		pruneFunc, ok := pruneSym.(func(string) ([]string, error))
		if !ok {
			fmt.Println(ok)
			panic("unexpected type from module symbol")
		}
		strings, err := pruneFunc("/mnt")
		fmt.Println(strings)

	} else {
		printError("you have to specicy either --set-expire-date or --prune")
	}
}
