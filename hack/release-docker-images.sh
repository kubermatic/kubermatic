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

### Builds and pushes all KKP Docker images:
###
### * quay.io/kubermatic/kubermatic[-ee]
### * quay.io/kubermatic/addons
### * quay.io/kubermatic/nodeport-proxy
### * quay.io/kubermatic/kubeletdnat-controller
### * quay.io/kubermatic/user-ssh-keys-agent
### * quay.io/kubermatic/etcd-launcher
### * quay.io/kubermatic/network-interface-manager
###
### The images are tagged with all arguments given to the script, i.e
### `./release-docker-images.sh foo bar` will tag `kubermatic:foo` and
### `kubermatic:bar`.
###
### Before running this script, all binaries in `cmd/` must have been
### built already by running `make build`.

set -euo pipefail

if [ "$#" -lt 1 ] || [ "${1}" == "--help" ]; then
  cat << EOF
Usage: $(basename $0) tag1[ tag2 tagN ...]

Example:
  $(basename $0) 0cf36a568b0911ac6688115df53c1f1701f45fcd6be5fc97fd6dc0410ac37a82 v2.5
EOF
  exit 0
fi

cd $(dirname "$0")/..
source hack/lib.sh

export ALL_TAGS=$@
export DOCKER_REPO="${DOCKER_REPO:-quay.io/kubermatic}"
export GOOS="${GOOS:-linux}"
export KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ee}"
export ARCHITECTURES="${ARCHITECTURES:-linux/amd64,linux/arm64/v8}"

REPOSUFFIX=""
if [ "$KUBERMATIC_EDITION" != "ce" ]; then
  REPOSUFFIX="-$KUBERMATIC_EDITION"
fi

# build Docker images
PRIMARY_TAG="${1}"
make docker-build TAGS="$PRIMARY_TAG"
make -C cmd/nodeport-proxy docker TAG="$PRIMARY_TAG"
docker build -t "$DOCKER_REPO/addons:$PRIMARY_TAG" addons
docker build -t "$DOCKER_REPO/etcd-launcher:$PRIMARY_TAG" -f cmd/etcd-launcher/Dockerfile .

# switch to a multi platform-enabled builder
docker buildx create --use

build_tag_flags() {
  local repository="$1"

  for tag in $ALL_TAGS; do
    if [ -z "$tag" ]; then
      continue
    fi

    echo -n " --tag $repository:$tag"
  done

  echo
}

buildx_build() {
  local context="$1"
  local file="$2"
  local repository="$3"

  docker buildx build \
    --push \
    --platform "$ARCHITECTURES" \
    --build-arg "GOPROXY=${GOPROXY:-}" \
    --build-arg "KUBERMATIC_EDITION=$KUBERMATIC_EDITION" \
    --provenance false \
    --file "$file" \
    $(build_tag_flags "$repository") \
    $context
}

# build and push multi-arch images
# (buildx cannot just build and load a multi-arch image,
# see https://github.com/docker/buildx/issues/59)
echodate "Building user-ssh-keys-agent images..."
buildx_build . cmd/user-ssh-keys-agent/Dockerfile.multiarch "$DOCKER_REPO/user-ssh-keys-agent"

echodate "Building kubeletdnat-controller images..."
buildx_build . cmd/kubeletdnat-controller/Dockerfile.multiarch "$DOCKER_REPO/kubeletdnat-controller"

echodate "Building network-interface-manager images..."
buildx_build . cmd/network-interface-manager/Dockerfile.multiarch "$DOCKER_REPO/network-interface-manager"

# for each given tag, tag and push the image
for TAG in "$@"; do
  if [ -z "$TAG" ]; then
    continue
  fi

  echodate "Tagging as $TAG"
  docker tag "$DOCKER_REPO/kubermatic$REPOSUFFIX:$PRIMARY_TAG" "$DOCKER_REPO/kubermatic$REPOSUFFIX:$TAG"
  docker tag "$DOCKER_REPO/nodeport-proxy:$PRIMARY_TAG" "$DOCKER_REPO/nodeport-proxy:$TAG"
  docker tag "$DOCKER_REPO/addons:$PRIMARY_TAG" "$DOCKER_REPO/addons:$TAG"
  docker tag "$DOCKER_REPO/etcd-launcher:$PRIMARY_TAG" "$DOCKER_REPO/etcd-launcher:$TAG"

  echodate "Pushing images"
  docker push "$DOCKER_REPO/kubermatic$REPOSUFFIX:$TAG"
  docker push "$DOCKER_REPO/nodeport-proxy:$TAG"
  docker push "$DOCKER_REPO/addons:$TAG"
  docker push "$DOCKER_REPO/etcd-launcher:$TAG"
done
