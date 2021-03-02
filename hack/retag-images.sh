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

### This script takes YAML on stdin and tries to find all references
### to Docker images. The found images will then be pulled, retagged
### with `$TARGET_REGISTRY` as their registry and pushed to said target
### registry.
###
### The purpose of this is to mirror all images used in a KKP setup
### to prewarm a local Docker registry, for example in offline setups.

set -euo pipefail

TARGET_REGISTRY=${TARGET_REGISTRY:-127.0.0.1:5000}

function retag {
  local image="$1"

  # trim registry
  local local_image="$(echo ${image} | cut -d/ -f1 --complement)"

  # split into name and tag
  local name="$(echo ${local_image} | cut -d: -f1)"
  local tag="$(echo ${local_image} | cut -d: -f2)"

  # build target image name
  local target_image="${TARGET_REGISTRY}/${name}:${tag}"

  echo -n "Retagging ${image} => ${target_image}"

  if curl -s --fail "http://${TARGET_REGISTRY}/v2/${name}/tags/list" | jq -e ".tags | index(\"${tag}\")" > /dev/null; then
    echo " skipping, exists already"
    return
  fi

  echo " ..."

  docker pull "${image}"
  docker tag "${image}" "${target_image}"
  docker push "${target_image}"

  echo "Done retagging ${image}"
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
