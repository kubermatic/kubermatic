#!/usr/bin/env bash

ver=v0.6.0

set -euox pipefail

docker build --no-cache --pull --build-arg "version=$ver" -t quay.io/kubermatic/shellcheck:$ver .
docker push quay.io/kubermatic/shellcheck:$ver
