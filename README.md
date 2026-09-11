# xpire - a tool to manage data expiration

xpire is a CLI tool that can store information about when data should expire and delete the data accordingly.
It aims to be a simple tool usable by humans or in scripts to handle deletion of obsolete data.
xpire itself is stateless, no daemon or database backend is required, all information in stored in the filesystems themselfs.

While the xpire binary is only the user interface,
the actual work is done by plugins which should enable
xpire to make use of filesystem specific structures like
subvolumes or snapshots to prevent expensive tree walks
during pruning.
Each plugin can decide where the expiry date is stored but the most common case
is to use the extended attribute `user.expire="YYYY-MM-DD HH:MM:SS"`.

## Usage

```
# set a expiration date
xpire --path /data/foo/ --set "2023-05-01 15:00:00"

# list all expiration dates
xpire --path /data --list

# recursively prune all expired data
xpire --path /data --prune
```

Be arware that you might need root priviledges depending on the plugin used.

## Building from source

Just run `make` and then the binary is available under `./xpire`.

## Supported filesystems

Until now plugins for the following filesystems are provided by this repository:

* `btrfs`
* `zfs`

See [./filesystems/](./filesystems/) for details about them.

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
