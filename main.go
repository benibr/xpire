package main

import (
	"errors"
	"fmt"
	"os"
	"plugin"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"google.golang.org/genproto/googleapis/type/datetime"
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

func loadPlugin(path string) plugin.Plugin {
	fsType, err := getFsType(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(RC_ERR_FS)
	}
	fmt.Println("detected filesystem:", fsType)
	pluginPath := fmt.Sprintf("./filesystems/%s.so", fsType)
	plugin, err := plugin.Open(pluginPath)
	if err != nil {
		panic(err)
	}
	return *plugin
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
		plugin := loadPlugin(args.Path)
		setSym, err := plugin.Lookup("SetExpireDate")
		if err != nil {
			panic(err)
		}
		setFunc, ok := setSym.(func(bool, error) (time.Time, string))
		if !ok {
			panic("unexpected type from module symbol")
		}
		ok, _ := setFunc(args.SetExpireDate, args.Path)
		if !ok {
			panic("cannot set expiry date")
		}

		os.Exit(RC_OK)

	} else if args.Prune {
		checkPath()
		plugin := loadPlugin(args.Path)
		pruneSym, err := plugin.Lookup("PruneExpiredSnapshots")
		if err != nil {
			panic(err)
		}
		pruneFunc, ok := pruneSym.(func(string) ([]string, error))
		if !ok {
			fmt.Println(ok)
			panic("unexpected type from module symbol")
		}
		fmt.Printf("pruning all expired snapshots in '%s'\n", args.Path)
		pruned, err := pruneFunc(args.Path)
		fmt.Print("Pruned snapshots:")
		for i, _ := range pruned {
			print(i)
		}

	} else {
		printError("you have to specicy either --set-expire-date or --prune")
	}
}
