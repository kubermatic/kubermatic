#!/usr/bin/env bash

SUFFIX=""
VERSION=1.1.4

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/util:${VERSION}${SUFFIX} .
docker push quay.io/kubermatic/util:${VERSION}${SUFFIX}
