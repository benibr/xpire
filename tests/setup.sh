#!/bin/bash


#### BTRFS

# prepare btrfs
mkdir -p ./mnt/btrfs
truncate --size 5G btrfs.img
mkfs.btrfs btrfs.img &>/dev/null
mount -v -o loop btrfs.img ./mnt/btrfs

btrfs subvolume create ./mnt/btrfs/subvolume
btrfs subvolume snapshot ./mnt/btrfs/subvolume ./mnt/btrfs/snapshot
chmod 777 ./mnt/btrfs -R
btrfs subvolume create ./mnt/btrfs/root-only
chmod 640 ./mnt/btrfs/root-only
chown root: ./mnt/btrfs/root-only

sudo mount -v -o "loop,subvol=subvolume" btrfs.img ./mnt/btrfs/subvolume-mount

btrfs subvolume create ./mnt/btrfs/wrong-time-format
setfattr ./mnt/btrfs/wrong-time-format/ -n user.expire -v "205-02 111"

mkdir -p mnt/btrfs/dir
