#!/bin/bash

#########
# BTRFS #
#########
# prepare btrfs
mkdir -p ./mnt/btrfs
truncate --size 5G btrfs.img
mkfs.btrfs btrfs.img &>/dev/null
mount -v -o loop btrfs.img ./mnt/btrfs

# regular subvolumes
btrfs subvolume create ./mnt/btrfs/subvolume01
btrfs subvolume create ./mnt/btrfs/subvolume02
btrfs subvolume create ./mnt/btrfs/subvolume03

# snapshots
btrfs subvolume snapshot ./mnt/btrfs/subvolume01 ./mnt/btrfs/snapshot

# default permissions
chmod 777 ./mnt/btrfs -R

# root only subvolume
btrfs subvolume create ./mnt/btrfs/root-only
chmod 640 ./mnt/btrfs/root-only
chown root: ./mnt/btrfs/root-only

# subvolume mount under other name
mkdir -p ./mnt/btrfs/subvolume-mount
sudo mount -v -o "loop,subvol=subvolume01" btrfs.img ./mnt/btrfs/subvolume-mount

# subvolume with wrong expire date
btrfs subvolume create ./mnt/btrfs/wrong-time-format
setfattr ./mnt/btrfs/wrong-time-format/ -n user.expire -v "205-02 111"

# non subvolumes
mkdir -p mnt/btrfs/dir
