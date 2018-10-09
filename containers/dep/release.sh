#!/usr/bin/env bash

NUMBER=1
VERSION=0.5.0


set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/dep:${VERSION}-${NUMBER} .
docker push quay.io/kubermatic/dep:${VERSION}-${NUMBER}
