#!/usr/bin/env bash

ver=v0.2-dev-2

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/curl:$ver .
docker push quay.io/kubermatic/curl:$ver
