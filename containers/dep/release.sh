#!/usr/bin/env bash

ver=0.5.0

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/dep:$ver .
docker push quay.io/kubermatic/dep:$ver
