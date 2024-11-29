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
### container images by using `hack/release-images.sh`. It is only
### useful as part of another script to setup KKP for testing.

set -eu

cd $(dirname $0)/../..
source hack/lib.sh

GIT_HEAD_HASH="$(git rev-parse HEAD)"
GIT_HEAD_TAG="$(git tag -l "$PULL_BASE_REF")"
GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"

# prepare special variables that will be injected into the Kubermatic Operator;
# use the latest tagged version of the dashboard when we ourselves are a tagged
# release
export KUBERMATICDOCKERTAG="${GIT_HEAD_TAG:-$GIT_HEAD_HASH}"

if [ -z "${UIDOCKERTAG:-}" ]; then
  export UIDOCKERTAG="$KUBERMATICDOCKERTAG"

  if [ -z "$GIT_HEAD_TAG" ]; then
    # in presubmits, we check against the PULL_BASE_REF (which is incidentally also
    # only defined in presubmits); in postsubmits we check against the current branch
    UIBRANCH="${PULL_BASE_REF:-$GIT_BRANCH}"

    # the dashboard only publishes container images for tagged releases and all
    # main branch revisions; this means for Kubermatic tests in release branches
    # we need to use the latest tagged dashboard of the same branch
    if [ "$UIBRANCH" == "main" ]; then
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
fi

# if no explicit (usually temporary) tags are given, default to tagging based on Git tags and hashes
if [ -z "${TAGS:-}" ]; then
  TAGS="$KUBERMATICDOCKERTAG"

  if [ "$GIT_BRANCH" == "main" ]; then
    # we only want to create the "latest" tag if we're building the main branch
    TAGS="$TAGS latest"
  elif [[ "$GIT_BRANCH" =~ release/v[0-9]+.* ]]; then
    # the dashboard e2e jobs in a release branch rely on a "latest" tag being
    # available for KKP, so we turn "release/v2.21" into "v2.21-latest"
    RELEASE_LATEST="${GIT_BRANCH#release/}"
    RELEASE_LATEST="${RELEASE_LATEST//\//-}-latest"

    TAGS="$TAGS $RELEASE_LATEST"
  fi
fi

if [ -z "${NO_IMAGES:-}" ]; then
  start_docker_daemon_ci

  if [ -z "${NO_PUSH:-}" ]; then
    echodate "Logging into Quay..."
    retry 5 buildah login -u "$QUAY_IO_USERNAME" -p "$QUAY_IO_PASSWORD" quay.io
    echodate "Successfully logged into Quay."
  else
    echodate "Skipping Quay login because \$NO_PUSH is set."
  fi
fi

echodate "Building and pushing container images…"

# prepare Helm charts, do not use $KUBERMATICDOCKERTAG for the chart version,
# as it could just be a git hash and we need to ensure a proper semver version
set_helm_charts_version "${GIT_HEAD_TAG:-9.9.9-$GIT_HEAD_HASH}" "$KUBERMATICDOCKERTAG"

# make sure that the main container image contains ready made CRDs, as the installer
# will take them from the operator chart and not use the compiled-in versions.
copy_crds_to_chart
set_crds_version_annotation

# finally build and push the container images that are KKP-version dependent (i.e.
# not http-prober or the util images)
images=(
  addons
  alertmanager-authorization-server
  conformance-tester
  etcd-launcher
  kubeletdnat-controller
  kubermatic
  network-interface-manager
  nodeport-proxy
  user-ssh-keys-agent
)

# TODO: Bring back the ability to set multiple tags at once.
export TAG="$KUBERMATICDOCKERTAG"
retry 1 ./hack/images/release.sh "${images[@]}"

echodate "Successfully finished building and pushing container images"
