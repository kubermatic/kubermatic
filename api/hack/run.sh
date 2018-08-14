#!/usr/bin/env bash
export REL_CODE_DIR="$(dirname "${0}")/../../"
export ABS_CODE_DIR=$(realpath ${REL_CODE_DIR})

docker run \
    -ti \
    -v ${ABS_CODE_DIR}:/go/src/github.com/kubermatic/kubermatic \
    -v $(go env GOCACHE):/root/.cache/go-build \
    quay.io/kubermatic/build:v0.1 "$*"
