package helpers

import (
	"os"
	"path/filepath"
)

// return a clean absolute path
func CleanPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absPath, err = filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", err
	}
	absPath = filepath.Clean(absPath)
	return absPath, nil
}

// check if xpire runs as root
func IsRoot() bool {
	uid := os.Getuid()
	return uid == 0
}
