package main

import (
	"time"
	//"github.com/pkg/xattr"
)

// internal functions


// mandatory functions called by fsexpire
func setExpireDate(date time.Time, path string) (bool, error) {
	return true, nil
}

func pruneExpiredSnapshots (path string) ([]string, error) {
	return nil, nil
}
