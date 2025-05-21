# xpire - a tool to manage data expiration

xpire uses extended attributes (xattrs) of filesystems to store information
about when a data should expire.
It aims to provide a simple to use interface for setting and changing these
dates and pruning all expired files.

While the xpire binary is only the user interface,
the actual work is done by plugins which should enable
xpire to make use of filesystem specific structures like
subvolumes or snapshots to prevent expensive tree walks
during pruning.

## Usage

Currently two main functionalies are provided, setting a expire date
and pruneing all expired files.
Be arware that you might need root priviledges.

```sh
$ xpire --path /path/to/old/data --set "2023-05-01 15:00:00"
INFO Detected filesystem: btrfs
INFO setting expiration date on snapshot '/path/to/old/data' to 2023-05-01 15:00:00
```

```sh
$ xpire --path /path --prune
INFO Detected filesystem: btrfs
INFO pruning all expired snapshots in 'mnt/btrfs'
INFO ↳ '/path/to/old/data' expired since 2023-05-01 15:00:00
```

## Building from source

Just run `make` and then the binary is available under `./xpire`.

## Supported filesystems

Until now plugins for the following filesystems are provided by this repository:

* `btrfs`

## Development

`xpire` is still under heavy development which means the either CLI parameters or
the plugin API may change without prior notice.
This tool and all plugins come as they are and without any warrenty.
Do not use for production data.
