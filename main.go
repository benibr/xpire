package main

import (
  "fmt"
  "os"
  "time"
  "errors"
  "syscall"
  "github.com/alexflint/go-arg"
)

const RC_OK = 0
const RC_ERR_ARGS = 5

var args struct {
	SetExpireDate string `arg:"-s,--set-expire-date"`
	Path string
	Prune bool
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
	if (args.SetExpireDate != "") {
		checkPath()
		if args.Prune {
			printError("Cannot use --prune with --setexpiredate")
			os.Exit(RC_ERR_ARGS)
		}
		parsedTime, err := time.Parse(time.DateTime, args.SetExpireDate)
		if err != nil {
			fmt.Println(err)
			os.Exit(5)
		}
		fmt.Printf("setting expiration date on snapshot '%s' to %s\n", args.Path, parsedTime.Format(time.DateTime))
		os.Exit(RC_OK)
	}

	// prune expired snapshots
	if args.Prune {
		checkPath()
		fmt.Printf("pruning all expired snapshots in '%s'\n", args.Path)
		fsType, err := getFsType(args.Path)
		if err != nil {
			fmt.Println(err)
			printError("cannot read specified path")
		}
		fmt.Println("detected filesystem:", fsType)
	}
}
