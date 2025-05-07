#!/bin/bash

# prepare btrfs
mkdir -p mnt/btrfs
truncate --size 5G btrfs.img
mkfs.btrfs btrfs.img
sudo mount btrfs.img mnt/btrfs

