#!/bin/bash

echo "#########"
echo "# BTRFS #"
echo "#########"
# prepare btrfs
mkdir -p ./mnt/btrfs
truncate --size 5G btrfs.img
mkfs.btrfs btrfs.img &>/dev/null
mount -v -o loop btrfs.img ./mnt/btrfs

# regular subvolumes
btrfs subvolume create ./mnt/btrfs/subvolume01
btrfs subvolume create ./mnt/btrfs/subvolume02
btrfs subvolume create ./mnt/btrfs/subvolume03
btrfs subvolume create ./mnt/btrfs/subvolume03/subvolume30
btrfs subvolume create ./mnt/btrfs/subvolume03/subvolume31
setfattr ./mnt/btrfs/subvolume03/subvolume30 -n user.expire -v "2002-01-01 15:00:00"
setfattr ./mnt/btrfs/subvolume03/subvolume31 -n user.expire -v "2002-01-01 15:00:00"

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





echo "#######"
echo "# ZFS #"
echo "#######"
# prepare zfs
mkdir -p ./mnt/zfs/
rm -f zfs.img
truncate --size 5G zfs.img
zpool create xpool -m $(readlink -f ./mnt/zfs) ./zfs.img -f
zfs create xpool/dataset00

# regular datasets
zfs create xpool/dataset00/dataset01
zfs create xpool/dataset00/dataset01/dataset11
zfs create xpool/dataset00/dataset02
setfattr ./mnt/zfs/dataset00/dataset02 -n user.expire -v "2002-01-01 15:00:00"
zfs create xpool/dataset00/dataset03
zfs create xpool/dataset00/dataset03/dataset33
setfattr ./mnt/zfs/dataset00/dataset03/dataset33 -n user.expire -v "2002-01-01 15:00:00"

# dataset with wrong expire date
zfs create xpool/dataset00/wrong-time-format
setfattr ./mnt/zfs/dataset00/wrong-time-format/ -n user.expire -v "205-02 111"

# non subvolumes
mkdir -p mnt/zfs/dir

# snapshots

# default permissions
chmod 777 ./mnt/zfs -R
