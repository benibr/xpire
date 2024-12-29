package main

import (
	"time"
	//"github.com/pkg/xattr"
)

// internal functions

// mandatory functions called by fsexpire
func SetExpireDate(date time.Time, path string) (bool, error) {
	return true, nil
}

func PruneExpiredSnapshots(path string) ([]string, error) {
	return nil, nil
}

func main() {}
