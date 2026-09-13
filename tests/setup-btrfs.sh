#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

echo "#########"
echo "# BTRFS #"
echo "#########"
# prepare btrfs
mkdir -p ./mnt/btrfs
truncate --size 5G btrfs.img
mkfs.btrfs btrfs.img &>/dev/null
mount -v -o loop btrfs.img ./mnt/btrfs
