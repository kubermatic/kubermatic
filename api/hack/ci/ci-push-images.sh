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

# inject current dashboard version into Kubermatic Operator;
# PULL_BASE_REF is the name of the current branch in case of a post-submit
# or the name of the base branch in case of a PR.
DASHBOARDCOMMIT="$(get_latest_dashboard_hash "${PULL_BASE_REF}")"

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
