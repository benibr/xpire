## v0.2.0

* added ZFS plugin

## v0.2.0

* `--loglevel` can be used to make xpire more/less verbose
* btrfs: checks for missing permissions
* btrfs: prune function now understands nested subvolume structures
* plugin API definition now available under `pluginapi/pluginapi.go`
* prepared code and folder structure to support bigger codebase
* xpire is now under GPL-3.0 license

## v0.1.1 - Initial release

This release contains the first working version of xpire including:

* CLI parameters for setting expire date
* CLI parameters for pruning expired data
* CLI parameter for explicit plugin selection
* btrfs Plugin which can set dates and delete subvolumes
* Logging to stderr
