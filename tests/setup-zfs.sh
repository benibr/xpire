#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"
DIR="${XPIRE_TEST_DIR:-$PWD}"

if ! command -v zpool >/dev/null; then
  echo "zpool not found, skipping ZFS setup" >&2
  exit 0
fi

if zpool list xpire-pool &>/dev/null; then
  echo "pool 'xpire-pool' already exists, run teardown-zfs.sh first" >&2
  exit 1
fi

echo "#######"
echo "# ZFS #"
echo "#######"
# prepare zfs
mkdir -p "$DIR/mnt/zfs"
chmod 755 "$DIR" "$DIR/mnt"
rm -f "$DIR/zfs.img"
truncate --size 5G "$DIR/zfs.img"
zpool create xpire-pool -m "$(readlink -f "$DIR/mnt/zfs")" "$DIR/zfs.img" -f
