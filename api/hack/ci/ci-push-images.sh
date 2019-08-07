#!/bin/sh

set -eu
cd "$(git rev-parse --show-toplevel)"

. ./api/hack/lib.sh

GIT_HEAD_HASH="$(git rev-parse HEAD)"
# FIXME: use `latest` only on the master branch
TAGS="$GIT_HEAD_HASH $(git tag -l --points-at HEAD) latest"

apt install time -y

echodate "Logging into Quay"
docker ps > /dev/null 2>&1 || start-docker.sh
retry 5 docker login -u "$QUAY_IO_USERNAME" -p "$QUAY_IO_PASSWORD" quay.io
echodate "Successfully logged into Quay"

echodate "Building binaries"
time make -C api build
echodate "Successfully finished building binaries"

echodate "Building and pushing quay images"
set -f # prevent globbing, do word splitting
# shellcheck disable=SC2086
retry 5 ./api/hack/push_image.sh $TAGS
echodate "Sucessfully finished building and pushing quay images"

echodate "Building addons"
time docker build -t "quay.io/kubermatic/addons:$GIT_HEAD_HASH" ./addons
for TAG in $TAGS; do
    [ -z "$TAG" ] && continue

    if ! [ "$TAG" = "$GIT_HEAD_HASH" ]; then
      echo "Tagging ${TAG}"
      docker tag "quay.io/kubermatic/addons:$GIT_HEAD_HASH" "quay.io/kubermatic/addons:$TAG"
    fi

    echo "Pushing ${TAG}"
    retry 5 docker push "quay.io/kubermatic/addons:$TAG"
done
echodate "Successfully finished building and pushing addon image"
