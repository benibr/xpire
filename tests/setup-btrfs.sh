#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

if ! command -v btrfs >/dev/null; then
  echo "btrfs not found, skipping btrfs setup" >&2
  exit 0
fi

echo "#########"
echo "# BTRFS #"
echo "#########"
# prepare btrfs
mkdir -p ./mnt/btrfs
truncate --size 5G btrfs.img
mkfs.btrfs btrfs.img &>/dev/null
mount -v -o loop btrfs.img ./mnt/btrfs
