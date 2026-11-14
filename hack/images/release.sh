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

set -euo pipefail

cd $(dirname $0)
source ../lib.sh

REPOSITORY="$1"
source "$REPOSITORY/settings.sh"

DOCKER_REPO="${DOCKER_REPO:-quay.io/kubermatic}"
DRY_RUN=${DRY_RUN:-true}
IMAGE="$DOCKER_REPO/$REPOSITORY:$TAG"
HEAD="$(git rev-parse HEAD)"

# images that set PLATFORMS in their settings.sh are built for several
# architectures with buildx; a multi-arch image cannot be loaded into the
# local docker daemon, so it is pushed straight from the builder
if [[ -n "${PLATFORMS:-}" ]]; then
  if ! docker buildx inspect kkp-utility-images > /dev/null 2>&1; then
    docker buildx create --name kkp-utility-images --driver docker-container > /dev/null
  fi

  docker buildx build \
    --builder kkp-utility-images \
    --no-cache \
    --pull \
    --platform "$PLATFORMS" \
    --tag "$IMAGE" \
    --label "org.opencontainers.image.version=$TAG" \
    --label "org.opencontainers.image.revision=$HEAD" \
    --output "type=image,push=$(if $DRY_RUN; then echo false; else echo true; fi)" \
    "$REPOSITORY"

  if $DRY_RUN; then
    echodate "Set \$DRY_RUN=false to push the image."
  fi

  exit 0
fi

docker build \
  --no-cache \
  --pull \
  --tag "$IMAGE" \
  --label "org.opencontainers.image.version=$TAG" \
  --label "org.opencontainers.image.revision=$HEAD" \
  "$REPOSITORY"

if $DRY_RUN; then
  echodate "Set \$DRY_RUN=false to push the image."
else
  docker push "$IMAGE"
fi
