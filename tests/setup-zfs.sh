#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

echo "#######"
echo "# ZFS #"
echo "#######"
# prepare zfs
mkdir -p ./mnt/zfs/
rm -f zfs.img
truncate --size 5G zfs.img
zpool create xpool -m $(readlink -f ./mnt/zfs) ./zfs.img -f
