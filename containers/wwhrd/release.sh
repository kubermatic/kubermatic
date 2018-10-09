#!/usr/bin/env bash

NUMBER=dev-1
VERSION=0.2.1

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/wwhrd:${VERSION}-${NUMBER} .
docker push quay.io/kubermatic/wwhrd:${VERSION}-${NUMBER}
