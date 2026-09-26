#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"
DIR="${XPIRE_TEST_DIR:-$PWD}"

if ! command -v btrfs >/dev/null; then
  echo "btrfs not found, skipping btrfs teardown" >&2
  exit 0
fi

echo "tearing down btrfs"
umount -l "$DIR/mnt/btrfs" 2>/dev/null || true
umount -l "$DIR/mnt/btrfs/subvolume-mount" 2>/dev/null || true
# a lazy umount keeps the loop device while files are still open,
# detach it so that the image can be deleted and recreated
losetup -j "$DIR/btrfs.img" -O NAME -n | xargs -r -n1 losetup -d
rm -f "$DIR/btrfs.img"
