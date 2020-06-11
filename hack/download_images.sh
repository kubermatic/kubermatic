#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/..
make image-loader
_build/image-loader -logtostderr=true --registry-name="test.registry" -print-only
