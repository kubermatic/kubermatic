#!/usr/bin/env bash

ver=v0.2

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/conformance-tests:${ver} .
docker push quay.io/kubermatic/conformance-tests:${ver}
