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
### In presubmits (DRY_RUN=true and PULL_BASE_SHA set), every image whose
### directory changed in the PR is built even if its tag exists, and the job
### fails if such a change does not come with a new TAG in settings.sh.

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

CHANGED_FILES=""
if $DRY_RUN && [[ -n "${PULL_BASE_SHA:-}" ]]; then
  CHANGED_FILES="$(git diff --name-only "$PULL_BASE_SHA...HEAD")"
fi

# tests next to a sidecar's code do not end up in its image
dir_changed() {
  local files="$CHANGED_FILES"
  if [[ "$1" == cmd/* ]]; then
    files="$(grep -v '_test\.go$' <<< "$files" || true)"
  fi
  grep -q "^$1/" <<< "$files"
}

# prints nothing if the directory does not exist on the PR base yet;
# the local TAG keeps the base value away from the caller's TAG
base_tag() {
  local TAG=""
  # eval instead of source <(...), which reads nothing on bash 3.2 (macOS)
  eval "$(git show "$PULL_BASE_SHA:$1/settings.sh" 2> /dev/null || true)"
  echo "$TAG"
}

# should_build decides if the image for directory $1 needs to be built.
# Outside of presubmits it only checks the registry. In presubmits a changed
# directory is always built, but must come with a TAG that is new compared to
# the PR base and not yet published, or the postsubmit would never push it.
should_build() {
  local dir="$1" name="$2" image="$3"

  if [[ -n "$CHANGED_FILES" ]] && dir_changed "$dir"; then
    if [[ "$(base_tag "$dir")" == "$TAG" ]]; then
      echodate "[$name] $dir/ changed, but TAG in $dir/settings.sh is still $TAG. Bump the TAG so the new image gets published after merging."
      exit 1
    fi

    if image_exists "$image"; then
      echodate "[$name] $dir/ changed, but $image exists already and would not be overwritten. Choose a new TAG in $dir/settings.sh."
      exit 1
    fi

    echodate "[$name] $dir/ changed in this PR, building $image."
    return 0
  fi

  if image_exists "$image"; then
    echodate "[$name] $image exists already, skipping."
    return 1
  fi

  return 0
}

if [ "$COMMANDS" != "false" ]; then
  for sidecar in $COMMANDS; do
    TEST_NAME="Build $sidecar image"
    echodate "[$sidecar] Checking…"

    source "cmd/$sidecar/settings.sh"
    IMAGE="$DOCKER_REPO/$sidecar:$TAG"

    if ! should_build "cmd/$sidecar" "$sidecar" "$IMAGE"; then
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
  imageDir="${imageDir%/}"
  imageName="$(basename "$imageDir")"

  TEST_NAME="Build $imageName image"
  echodate "[$imageName] Checking…"

  source "$imageDir/settings.sh"
  IMAGE="$DOCKER_REPO/$imageName:$TAG"

  if ! should_build "$imageDir" "$imageName" "$IMAGE"; then
    continue
  fi

  echodate "[$imageName] Building $IMAGE…"
  hack/images/release.sh "$imageName"
done

echodate "Done."
