# xpire - a tool to manage data expiration

xpire uses extended attributes (xattrs) of filesystems to store information
about when data should expire.
It aims to provide a simple to use interface for setting and changing these
dates and pruning expired files.

While the xpire binary is only the user interface,
the actual work is done by plugins which should enable
xpire to make use of filesystem specific structures like
subvolumes or snapshots to prevent expensive tree walks
during pruning.

xpire uses the extended attribute `user.expire="YYYY-MM-DD HH:MM:SS"`.

## Usage

Currently two main functionalities are provided: **setting a expire date**
and **pruneing all expired files**.
Be arware that you might need root priviledges.

```sh
$ xpire --path /path/to/old/data --set "2023-05-01 15:00:00"
INFO Detected filesystem: btrfs
INFO setting expiration date on '/path/to/old/data' to 2023-05-01 15:00:00
```

```sh
$ xpire --path /path --prune
INFO Detected filesystem: btrfs
INFO pruning all expired data in '/path'
INFO ↳ '/path/to/old/data' expired since 2023-05-01 15:00:00
```

## Building from source

Just run `make` and then the binary is available under `./xpire`.

## Supported filesystems

Until now plugins for the following filesystems are provided by this repository:

* `btrfs`
* `zfs`

## Development status

`xpire` is still under heavy development which means that both CLI parameters and
the plugin API may change without prior notice before version `1.0` is reached.
This tool and all plugins come as they are and without any warranty.
Do not use for production data.

## Contribution

Feedback and contributions are welcome! Please use
[Github issues](https://github.com/benibr/xpire/issues) and
[Github pull requests](https://github.com/benibr/xpire/pulls).

For writing new plugins, take a look at [./pluginapi/pluginapi.go](./pluginapi/pluginapi.go)
first.
