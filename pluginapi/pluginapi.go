package pluginapi

import (
	"github.com/sirupsen/logrus"
	"time"
)

// This interface defines the mandatory functions every filesystem
// plugin must implement.

type FsPluginApi interface {
	// InitLogger gets called right after plugin initialization
	// to pass the current logrus instance point to the plugin
	InitLogger(l *logrus.Logger) error

	// SetExpireDate is used to set the expire date on a given
	// file/folder/subset.
	// Must overwrite the date without asking in case it's already set.
	// Return only errors
	SetExpireDate(t time.Time, path string) error

	// UnsetExpireDate is used to remove a expire date on a given
	// file/folder/subset.
	// Must overwrite the date without asking in case it's already set.
	// Return only errors
	UnsetExpireDate(path string) error

	// PruneExpired cleans up all expired files/folders/subsets
	// Gets a list of absoulte paths that have a expire date set (eg. found by List)
	// Must check if a path is expired or not
	// Must check/handle permissions.
	// Returns only errors.
	PruneExpired(paths []string) error

	// List all expire dates that are currently set
	// regardless if they're reached or not
	// Returns a list of all absolute paths that have
	// a expire date set.
	List(path string) ([]string, error)
}
