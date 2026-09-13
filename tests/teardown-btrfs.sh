#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

echo "tearing down btrfs"
umount -l ./mnt/btrfs 2>/dev/null || true
umount -l ./mnt/btrfs/subvolume-mount 2>/dev/null || true
rm -f btrfs.img
