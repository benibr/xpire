package main

import (
  "fmt"
  "os"
  "time"
  "github.com/alexflint/go-arg"
)

const RC_OK = 0
const RC_ERR_ARGS = 5

var args struct {
	SetExpireDate string
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
		fmt.Printf("setting expiration date on snapshot '%s' to %s", args.Path, parsedTime.Format(time.DateTime))
		os.Exit(RC_OK)
	}

	// prune expired snapshots
	if args.Prune {
		checkPath()
		fmt.Printf("pruning all expired snapshots in '%s'", args.Path)
	}
}
