# xpire filesystem plugin for btrfs

This plugin allows to set expire dates on [btrfs](https://de.wikipedia.org/wiki/Btrfs)
subvolumes usind xattrs.
During pruning it looks for all subvolumes (incl. snapshots) under the given path
and removes all that have a passed date set.
