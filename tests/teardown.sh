#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

./teardown-btrfs.sh
./teardown-zfs.sh

for i in ./mnt/*; do
  umount -l "$i" 2>/dev/null || true
done

rm -rf ./mnt *.img
