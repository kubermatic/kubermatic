#!/usr/bin/env bash

set -euo pipefail
export GIT_HEAD_HASH="$(git rev-parse HEAD|tr -d '\n')"
cd $(dirname $0)/../../..

source ./api/hack/lib.sh

echodate "Logging into Quay"
docker ps &>/dev/null || start-docker.sh
retry 5 docker login -u ${QUAY_IO_USERNAME} -p ${QUAY_IO_PASSWORD} quay.io
echodate "Successfully logged into Quay"

echodate "Building binaries"
time make -C api build
echodate "Successfully finished building binaries"

echodate "Building and pushing quay images"
retry 5 ./api/hack/push_image.sh $GIT_HEAD_HASH $(git tag -l --points-at HEAD)
echodate "Sucessfully finished building and pushing quay images"
