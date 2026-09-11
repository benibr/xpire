#!/bin/bash

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

# default permissions
chmod 777 ./mnt/zfs -R
