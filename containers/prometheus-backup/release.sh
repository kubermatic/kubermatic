#!/usr/bin/env bash

NUMBER=1
VERSION=1.0.0

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/prometheus-backup:${VERSION}-${NUMBER} .
docker push quay.io/kubermatic/prometheus-backup:${VERSION}-${NUMBER}
