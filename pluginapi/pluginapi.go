package pluginapi

import (
	"time"

	"github.com/sirupsen/logrus"
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
	// Returns only errors
	SetExpireDate(t time.Time, path string) error

	// UnsetExpireDate is used to remove a expire date on a given
	// file/folder/subset.
	// Returns only errors
	UnsetExpireDate(path string) error

	// PruneExpired cleans up all expired files/folders/subsets
	// Gets a map of absolute paths & their expiration dates.
	// xpire passes the return map from List to this function
	// to ensure the users delete only what they could also see
	// Must check if a path is expired or not
	// Must check/handle permissions.
	// Returns only errors.
	PruneExpired(paths map[string]time.Time) error

	// List all expire dates that are currently set
	// regardless if they're reached or not
	// Returns a map of absolute paths with their expiration dates
	List(path string) (map[string]time.Time, error)
}
