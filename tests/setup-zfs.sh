#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

if ! command -v zpool >/dev/null; then
  echo "zpool not found, skipping ZFS setup" >&2
  exit 0
fi

echo "#######"
echo "# ZFS #"
echo "#######"
# prepare zfs
mkdir -p ./mnt/zfs/
rm -f zfs.img
truncate --size 5G zfs.img
zpool create xpool -m $(readlink -f ./mnt/zfs) ./zfs.img -f
