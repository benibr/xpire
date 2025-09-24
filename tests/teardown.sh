#!/bin/bash

for i in ./mnt/*; do
  umount -l "$i"
done

rm -rf ./mnt *.img

zpool destroy xpool
