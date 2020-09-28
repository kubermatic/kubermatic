#!/usr/bin/env bash

# Copyright 2020 The Kubermatic Kubernetes Platform contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

### This script compiles the KKP binaries and then builds and pushes all
### Docker images by using `hack/release-docker-images.sh`. It is only
### useful as part of another script to setup KKP for testing.

set -eu

cd $(dirname $0)/../..
source hack/lib.sh

GIT_HEAD_HASH="$(git rev-parse HEAD)"
GIT_HEAD_TAG="$(git tag -l "$PULL_BASE_REF")"
GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
TAGS="$GIT_HEAD_HASH $GIT_HEAD_TAG"

# we only want to create the "latest" tag if we're building
# the master branch and it's not a canary deployment
if [ -z "${CANARY_DEPLOYMENT:-}" ] && [ "$GIT_BRANCH" == "master" ]; then
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
  # in presubmits, we check against the PULL_BASE_REF (which is incidently also
  # only defined in presubmits); in postsubmits we check against the current branch
  UIBRANCH="${PULL_BASE_REF:-$GIT_BRANCH}"

  # the dasboard only publishes Docker images for tagged releases and all
  # master branch revisions; this means for Kubermatic tests in release branches
  # we need to use the latest tagged dashboard of the same branch
  if [ "$UIBRANCH" == "master" ]; then
    UIDOCKERTAG="$(get_latest_dashboard_hash "${UIBRANCH}")"
  else
    UIDOCKERTAG="$(get_latest_dashboard_tag "${UIBRANCH}")"
  fi
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
retry 1 make build
echodate "Successfully finished building binaries"

TEST_NAME="Build and push docker images"
echodate "Building and pushing quay images"

# prepare Helm charts
sed -i "s/__KUBERMATIC_TAG__/${KUBERMATICDOCKERTAG}/g" charts/*/*.yaml
sed -i "s/__DASHBOARD_TAG__/$UIDOCKERTAG/g" charts/*/*.yaml

set -f # prevent globbing, do word splitting
# shellcheck disable=SC2086
retry 5 ./hack/release-docker-images.sh $TAGS
echodate "Sucessfully finished building and pushing quay images"
unset TEST_NAME
