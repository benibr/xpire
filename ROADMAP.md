### version 0.1:

- [x] rename to xpire
- [x] btrfs: prune with find all subvolumes
- [x] btrfs: warn if not a subvolume or snapshot
- [x] parameter: -p plugin selection
- [x] logging to stderr
- [x] concept of error handling
- [x] beautify parameter
- [x] code cleanup, plugin function lookup
- [x] README with usage
- [x] Github release

### version 0.2

- [x] btrfs: check for permissions
- [x] `--loglevel`
- [x] move functions out of main.go to seperate file
- [x] plugin interface definition
- [x] add license
- [x] one subfolder per plugin with READMEs
- [x] README for btrfs

### version 0.3

- [x] zfs plugin

### version 0.4

- [ ] add --list option
- [ ] add --unset option
- [ ] add --dry-run option
- [ ] tests in containers
- [ ] tests with multiple users
- [ ] check if more tests are needed

### version 0.5

- [ ] posix plugin

## version 0.6

- [x] Changelog workflow not depending on Github
- [ ] use /usr/lib/modules/6.16.8-arch3-1/build/include/uapi/linux/magic.h to autodetect filesystems

## version 0.7


## version 0.8 - going weird

- [ ] check if any kind of S3 plugin is feasible
- [ ] daos plugin
OR
- [ ] gpfs plugin


### version 0.9

- Allow plugin path to be $CWD or /usr
- [ ] Make install
- [ ] Make uninstall
- [ ] Containerfile
- [ ] AUR
- [ ] .rpm
- [ ] .deb

### version 1.0

- [ ] github pipeline to build all packages
- [ ] remove all #FIXMEs
- [ ] align log/error messages in all plugins

### Future Ideas

- [ ] database plugin?
