#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

if ! command -v btrfs >/dev/null; then
  echo "btrfs not found, skipping btrfs teardown" >&2
  exit 0
fi

echo "tearing down btrfs"
umount -l ./mnt/btrfs 2>/dev/null || true
umount -l ./mnt/btrfs/subvolume-mount 2>/dev/null || true
rm -f btrfs.img
