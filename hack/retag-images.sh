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

TARGET_REGISTRY=${TARGET_REGISTRY:-127.0.0.1:5000}

function retag {
  local IMAGE="$1"

  ORG="$(echo ${IMAGE} | cut -d / -f2)"
  NAME="$(echo ${IMAGE} | cut -d / -f3 | cut -d : -f1)"
  TAG="$(echo ${IMAGE} | cut -d / -f3 | cut -d : -f2)"
  TARGET_IMAGE="${TARGET_REGISTRY}/${ORG}/${NAME}:${TAG}"

  echo -n "Retagging ${IMAGE} => ${TARGET_IMAGE}"

  if curl -s --fail "http://${TARGET_REGISTRY}/v2/${ORG}/${NAME}/tags/list" | jq -e ".tags | index(\"${TAG}\")" >/dev/null; then
    echo " skipping, exists already"
    return
  fi

  echo " ..."

  docker pull "${IMAGE}"
  docker tag "${IMAGE}" "${TARGET_IMAGE}"
  docker push "${TARGET_IMAGE}"

  echo "Done retagging ${IMAGE}"
}

IMAGES=$(cat /dev/stdin | (grep "image: " || true) | cut -d : -f 2,3)
for IMAGE in ${IMAGES}; do
  # Make sure we strip all quotes
  IMAGE="${IMAGE%\'}"
  IMAGE="${IMAGE#\'}"
  IMAGE="${IMAGE%\"}"
  IMAGE="${IMAGE#\"}"
  retag "${IMAGE}"
done
