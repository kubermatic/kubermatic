#!/usr/bin/env bash

set -euo pipefail

# docker.io/busybox

cd $(dirname $0)/..
make image-loader

grep -R 'Image:' pkg/ |_build/image-loader -logtostderr=true --registry-name="test.registry"
