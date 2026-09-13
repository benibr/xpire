#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"
DIR="${XPIRE_TEST_DIR:-$PWD}"

./teardown-btrfs.sh
./teardown-zfs.sh

for i in "$DIR"/mnt/*; do
  umount -l "$i" 2>/dev/null || true
done

rm -rf "${DIR:?}/mnt" "${DIR:?}"/*.img
