#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

echo "tearing down zfs"
zpool destroy xpool 2>/dev/null || true
