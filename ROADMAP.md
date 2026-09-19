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
- [x] add --list option
- [x] add --unset option
- [x] fix existing tests (agent)
- [x] split tests per plugin (agent)
- [x] add unit tests (agent)
- [x] tests with multiple users. exec as nobody (agent)
- [ ] check if more tests are needed (agent)
- [x] add gitlab pipeline on PRs (agent)
- [x] protect main branch PRs (agent)

### version 0.5 - usability
- [ ] add --dry-run option
- [ ] consitent error format
- [ ] use /usr/lib/modules/6.16.8-arch3-1/build/include/uapi/linux/magic.h to autodetect filesystems
- [ ] make output more readable
  - [ ] only use absolute paths in output
- [ ] align log/error messages in all plugins

### version 0.5.y - packaging
- [ ] Allow plugin path to be $CWD or /usr
- [ ] Make install
- [ ] Make uninstall
- [ ] Containerfile
- [ ] AUR
- [ ] .rpm
- [ ] .deb
- [ ] github pipeline to build all packages (agent)

### version 0.5.z - developing
- [ ] evaluate other logger
- [ ] prune should return list of deleted paths and xpire shold print them
- [ ] only use `expiration_date` or `date` for short

### version 0.5 - basic plugin
- [ ] posix plugin
- [ ] LVM plugin

## version 0.6 - going weird
Add some excentric stuff, implement one of the following

- [ ] check if any kind of S3 plugin is feasible
- [ ] daos plugin
- [ ] gpfs plugin

### version 1.0 - stability
- [ ] remove all #FIXMEs


### Future Ideas
Just a collection of ideas that are not planned for impelmentation

- database plugin to delete data
- database plugin to store expiration
