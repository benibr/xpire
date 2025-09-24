# xpire filesystem plugin for zfs

This plugin allows to set expire dates on [zfs](https://en.wikipedia.org/wiki/ZFS)
datasets using xattrs.
During pruning it looks for all datasets (incl. snapshots) under the given path
and removes all that have a passed date set.
