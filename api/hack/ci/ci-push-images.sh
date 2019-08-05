#!/usr/bin/env bash

set -euo pipefail
export GIT_HEAD_HASH="$(git rev-parse HEAD|tr -d '\n')"
cd $(dirname $0)/../../..

source ./api/hack/lib.sh

TAGS=("$GIT_HEAD_HASH" "$(git tag -l --points-at HEAD)")

echodate "Logging into Quay"
docker ps &>/dev/null || start-docker.sh
retry 5 docker login -u ${QUAY_IO_USERNAME} -p ${QUAY_IO_PASSWORD} quay.io
echodate "Successfully logged into Quay"

echodate "Building binaries"
time make -C api build
echodate "Successfully finished building binaries"

echodate "Building and pushing quay images"
retry 5 ./api/hack/push_image.sh "${TAGS[@]}"
echodate "Sucessfully finished building and pushing quay images"

echodate "Building addons"
time docker build -t quay.io/kubermatic/addons:${GIT_HEAD_HASH} ./addons
for TAG in "${TAGS[@]}"
do
    if [ -z "$TAG" ]; then
      continue
    fi

    if [ "$TAG" = "$GIT_HEAD_HASH" ]; then
      continue
    fi

    echo "Tagging ${TAG}"
    docker tag quay.io/kubermatic/addons:${GIT_HEAD_HASH} quay.io/kubermatic/addons:${TAG}
done
echodate "Successfully finished building addon image"

echodate "Pushing addon images"
for TAG in "${TAGS[@]}"
do
    echo "Pusing ${TAG}"
    retry 5 docker push quay.io/kubermatic/addons:${TAG}
done
echodate "Successfully finished pusing addon images"