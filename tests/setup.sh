#!/bin/bash

# prepare btrfs
mkdir -p ./mnt/btrfs
truncate --size 5G btrfs.img
mkfs.btrfs btrfs.img &>/dev/null
sudo mount btrfs.img ./mnt/btrfs
sudo btrfs subvolume create ./mnt/btrfs/subvolume
sudo btrfs subvolume snapshot ./mnt/btrfs/subvolume ./mnt/btrfs/snapshot
sudo chmod 777 ./mnt/btrfs -R
mkdir -p mnt/btrfs/dir
