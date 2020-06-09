#!/usr/bin/env bash

set -euo pipefail

VERSION=1.1.0

docker build --no-cache --pull -t quay.io/kubermatic/grafana-plugins:${VERSION} .
docker push quay.io/kubermatic/grafana-plugins:${VERSION}
