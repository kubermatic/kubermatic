#!/usr/bin/env bash

ver=v0.1-dev1

set -euox pipefail

GOOS=linux GOARCH=amd64 go build github.com/kubermatic/kubermatic/api/cmd/conformance-tests

docker build --no-cache --pull -t quay.io/kubermatic/conformance-tests:${ver} .
docker push quay.io/kubermatic/conformance-tests:${ver}

rm conformance-tests
