#!/usr/bin/env bash

# Copyright 2024 The Kubermatic Kubernetes Platform contributors.
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

### This script builds all the utility container images required by or provided
### in KKP, like the http-prober, S3 exporter, util etc.
### These utilities are not versioned together with KKP and so this script will
### only build and push images if they are not yet present in the target registry.

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

export DOCKER_REPO="${DOCKER_REPO:-quay.io/kubermatic}"
export DRY_RUN=${DRY_RUN:-true}
COMMANDS="${COMMANDS:-alertmanager-authorization-server http-prober s3-exporter}"
HEAD="$(git rev-parse HEAD)"

if [[ -n "${PROW_JOB_ID:-}" ]]; then
  start_docker_daemon_ci

  if ! $DRY_RUN; then
    echodate "Logging into Quay…"
    retry 5 docker login -u "$QUAY_IO_USERNAME" -p "$QUAY_IO_PASSWORD" quay.io
  fi
fi

image_exists() {
  skopeo inspect "docker://$1" --no-tags > /dev/null 2>&1
}

if [ "$COMMANDS" != "false" ]; then
  for sidecar in $COMMANDS; do
    TEST_NAME="Build $sidecar image"
    echodate "[$sidecar] Checking…"

    source "cmd/$sidecar/settings.sh"
    IMAGE="$DOCKER_REPO/$sidecar:$TAG"

    if image_exists "$IMAGE"; then
      echodate "[$sidecar] $IMAGE exists already, skipping."
      continue
    fi

    echodate "[$sidecar] Building binary…"
    GOOS=linux GOARCH=amd64 make clean $sidecar

    echodate "[$sidecar] Building $IMAGE…"
    docker build \
      --no-cache \
      --pull \
      --tag "$IMAGE" \
      --label "org.opencontainers.image.version=$TAG" \
      --label "org.opencontainers.image.revision=$HEAD" \
      --file "cmd/$sidecar/Dockerfile" \
      .

    if $DRY_RUN; then
      echodate "[$sidecar] dry run requested, not pushing image."
    else
      echodate "[$sidecar] Pushing $IMAGE…"
      docker push "$IMAGE"
    fi
  done
fi

for imageDir in hack/images/*/; do
  imageName="$(basename "$imageDir")"

  TEST_NAME="Build $imageName image"
  echodate "[$imageName] Checking…"

  source "$imageDir/settings.sh"
  IMAGE="$DOCKER_REPO/$imageName:$TAG"

  if image_exists "$IMAGE"; then
    echodate "[$imageName] $IMAGE exists already, skipping."
    continue
  fi

  echodate "[$imageName] Building $IMAGE…"
  hack/images/release.sh "$imageName"
done

echodate "Done."
