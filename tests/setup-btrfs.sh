#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"
DIR="${XPIRE_TEST_DIR:-$PWD}"

if ! command -v btrfs >/dev/null; then
  echo "btrfs not found, skipping btrfs setup" >&2
  exit 0
fi

if mountpoint -q "$DIR/mnt/btrfs"; then
  echo "'$DIR/mnt/btrfs' is already mounted, run teardown-btrfs.sh first" >&2
  exit 1
fi

echo "#########"
echo "# BTRFS #"
echo "#########"
# prepare btrfs
mkdir -p "$DIR/mnt/btrfs"
chmod 755 "$DIR" "$DIR/mnt"
# detach loop devices of a previous run that a lazy umount left behind
losetup -j "$DIR/btrfs.img" -O NAME -n | xargs -r -n1 losetup -d
rm -f "$DIR/btrfs.img"
truncate --size 5G "$DIR/btrfs.img"
mkfs.btrfs -f "$DIR/btrfs.img" >/dev/null
mount -v -o loop "$DIR/btrfs.img" "$DIR/mnt/btrfs"
