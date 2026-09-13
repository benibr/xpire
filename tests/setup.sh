#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

./setup-btrfs.sh
./setup-zfs.sh
