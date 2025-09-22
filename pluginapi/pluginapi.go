package pluginapi

import (
	"github.com/sirupsen/logrus"
	"time"
)

// This interface defines the mandatory functions every filesystem
// plugin must implement.

type FsPluginApi interface {
	// InitLogger gets called right after plugin initialisation
	// to pass the current logrus instance point to the plugin
	InitLogger(l *logrus.Logger) error

	// SetExpireDate is used to set the expiry date  on a given
	// file/folder/subset. The function must overwrite the date
	// without asking in case it's already set.
	// Permission checks must be done in plugin.
	SetExpireDate(t time.Time, path string) error

	// PruneExpired cleans up all expired files/folders/subsets
	// under the given path recursivly without any further user interaction.
	// Permission checks must be done in plugin.
	PruneExpired(path string) ([]string, error)
}
