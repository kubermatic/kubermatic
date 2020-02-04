#!/bin/bash

set -eu
cd "$(git rev-parse --show-toplevel)"

. ./api/hack/lib.sh

GIT_HEAD_HASH="$(git rev-parse HEAD)"
GIT_HEAD_TAG="$(git tag -l --points-at HEAD)"
TAGS="$GIT_HEAD_HASH $GIT_HEAD_TAG"

# we only want to create the "latest" tag if we're building
# the master branch and it's not a canary deployment
if [ -z "${CANARY_DEPLOYMENT:-}" ] && [ "$(git rev-parse --abbrev-ref HEAD)" == "master" ]; then
  TAGS="$TAGS latest"
fi

apt install time -y

echodate "Logging into Quay"
docker ps > /dev/null 2>&1 || start-docker.sh
retry 5 docker login -u "$QUAY_IO_USERNAME" -p "$QUAY_IO_PASSWORD" quay.io
echodate "Successfully logged into Quay"

# prepare special variables that will be injected into the Kubermatic Operator;
# use the latest tagged version of the dashboard when we ourselves are a tagged
# release
export KUBERMATICDOCKERTAG="${GIT_HEAD_TAG:-$GIT_HEAD_HASH}"
export UIDOCKERTAG="$KUBERMATICDOCKERTAG"

if [ -z "$GIT_HEAD_TAG" ]; then
  UIDOCKERTAG="$(get_latest_dashboard_hash "${PULL_BASE_REF}")"
else
  if [ -z "$(check_dashboard_tag "$GIT_HEAD_TAG")" ]; then
    echo "Kubermatic was tagged as $GIT_HEAD_TAG, but this tag does not exist for the dashboard."
    echo "Please release a new version for the dashboard and re-run this job."
    exit 1
  fi
fi

TEST_NAME="Build binaries"
echodate "Building binaries"
# Retry is used to get the junit wrapping
retry 1 make -C api build
echodate "Successfully finished building binaries"

TEST_NAME="Build and push docker images"
echodate "Building and pushing quay images"
set -f # prevent globbing, do word splitting
# shellcheck disable=SC2086
retry 5 ./api/hack/push_image.sh $TAGS
echodate "Sucessfully finished building and pushing quay images"
unset TEST_NAME
