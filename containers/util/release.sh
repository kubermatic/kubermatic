#!/usr/bin/env bash

SUFFIX=""
VERSION=1.3.3-dev

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/util:${VERSION}${SUFFIX} .
docker push quay.io/kubermatic/util:${VERSION}${SUFFIX}
