#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"
DIR="${XPIRE_TEST_DIR:-$PWD}"

if ! command -v btrfs >/dev/null; then
  echo "btrfs not found, skipping btrfs setup" >&2
  exit 0
fi

echo "#########"
echo "# BTRFS #"
echo "#########"
# prepare btrfs
mkdir -p "$DIR/mnt/btrfs"
chmod 755 "$DIR" "$DIR/mnt"
truncate --size 5G "$DIR/btrfs.img"
mkfs.btrfs "$DIR/btrfs.img" &>/dev/null
mount -v -o loop "$DIR/btrfs.img" "$DIR/mnt/btrfs"
