#!/usr/bin/env bash

NUMBER=2
VERSION=2.11.0

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/helm:${VERSION}-${NUMBER} .
docker push quay.io/kubermatic/helm:${VERSION}-${NUMBER}
