#!/usr/bin/env bash

NUMBER=3
VERSION=18.09.0

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/docker:${VERSION}-${NUMBER} .
docker push quay.io/kubermatic/docker:${VERSION}-${NUMBER}
