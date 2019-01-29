#!/usr/bin/env bash

NUMBER=2
VERSION=2.7.0

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/promtool:${VERSION}-${NUMBER} .
docker push quay.io/kubermatic/promtool:${VERSION}-${NUMBER}
