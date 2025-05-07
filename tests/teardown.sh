#!/bin/bash

for i in ./mnt/*; do
  sudo umount -l "$i"
done

rm -rf ./mnt *.img
