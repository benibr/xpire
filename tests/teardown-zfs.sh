#!/bin/bash

echo "tearing down zfs"
zpool destroy xpool 2>/dev/null || true
