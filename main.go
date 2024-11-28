package main

import (
  "fmt"
//  "strconv"
  "os"
  "github.com/alexflint/go-arg"
)

const RC_OK = 0
const RC_ERR_ARGS = 5

var args struct {
	Expire string
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

func main() {

	arg.MustParse(&args)

	// set expiration date
	if (args.Expire != ""){
		checkPath()
		if args.Prune {
			printError("Cannot use --prune with --setexpiredate")
			os.Exit(RC_ERR_ARGS)
		}
		fmt.Printf("setting expiration date on snapshot '%s' to %s", args.Path, args.Expire)
		os.Exit(RC_OK)
	}

	if args.Prune {
		checkPath()
		fmt.Printf("pruning all expired snapshots in '%s'", args.Path)
	}
}
