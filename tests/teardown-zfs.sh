#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

if ! command -v zpool >/dev/null; then
  echo "zpool not found, skipping ZFS teardown" >&2
  exit 0
fi

echo "tearing down zfs"
zpool destroy xpire-pool 2>/dev/null || true
