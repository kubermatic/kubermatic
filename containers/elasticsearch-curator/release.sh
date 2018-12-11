#!/usr/bin/env bash

NUMBER=1
VERSION=5.6.0

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/elasticsearch-curator:${VERSION}-${NUMBER} .
docker push quay.io/kubermatic/elasticsearch-curator:${VERSION}-${NUMBER}
