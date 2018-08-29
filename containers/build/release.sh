#!/usr/bin/env bash

ver=v0.0.3

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/build:$ver .
docker push quay.io/kubermatic/build:$ver
