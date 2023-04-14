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

export DOCKER_REPO="${DOCKER_REPO:-quay.io/kubermatic}"
export GOOS="${GOOS:-linux}"
export ARCHITECTURES=${ARCHITECTURES:-amd64 arm64}

# build Docker images
PRIMARY_TAG="${1}"
make docker-build TAGS="${PRIMARY_TAG}"
make -C cmd/nodeport-proxy docker TAG="${PRIMARY_TAG}"
docker build -t "${DOCKER_REPO}/addons:${PRIMARY_TAG}" addons
docker build -t "${DOCKER_REPO}/etcd-launcher:${PRIMARY_TAG}" -f cmd/etcd-launcher/Dockerfile .

# prepare gocaches for each arch
gocaches="$(mktemp -d)"
for ARCH in ${ARCHITECTURES}; do
  cacheDir="$gocaches/$ARCH"
  mkdir -p "$cacheDir"

  # amd64 has been downloaded into $GOCACHE already, do not download it again
  if [ "$ARCH" == "amd64" ]; then
    cp -ar "$(go env GOCACHE)"/* "$cacheDir"
    continue
  fi

  # try to get a gocache for this arch; this can "fail" but still exit with 0
  TARGET_DIRECTORY="$cacheDir" GOARCH="$ARCH" ./hack/ci/download-gocache.sh
done

# build multi-arch images
buildah manifest create "${DOCKER_REPO}/user-ssh-keys-agent:${PRIMARY_TAG}"
for ARCH in ${ARCHITECTURES}; do
  echodate "Building user-ssh-keys-agent image for $ARCH..."

  buildah bud \
    --tag "${DOCKER_REPO}/user-ssh-keys-agent-${ARCH}:${PRIMARY_TAG}" \
    --build-arg "GOPROXY=${GOPROXY:-}" \
    --build-arg "GOCACHE=/gocache" \
    --arch "$ARCH" \
    --override-arch "$ARCH" \
    --format=docker \
    --file cmd/user-ssh-keys-agent/Dockerfile.multiarch \
    --volume "$gocaches/$ARCH:/gocache" \
    .
  buildah manifest add "${DOCKER_REPO}/user-ssh-keys-agent:${PRIMARY_TAG}" "${DOCKER_REPO}/user-ssh-keys-agent-${ARCH}:${PRIMARY_TAG}"
done

buildah manifest create "${DOCKER_REPO}/kubeletdnat-controller:${PRIMARY_TAG}"
for ARCH in ${ARCHITECTURES}; do
  echodate "Building kubeletdnat-controller image for $ARCH..."

  buildah bud \
    --tag "${DOCKER_REPO}/kubeletdnat-controller-${ARCH}:${PRIMARY_TAG}" \
    --build-arg "GOPROXY=${GOPROXY:-}" \
    --build-arg "GOCACHE=/gocache" \
    --arch "$ARCH" \
    --override-arch "$ARCH" \
    --format=docker \
    --file cmd/kubeletdnat-controller/Dockerfile.multiarch \
    --volume "$gocaches/$ARCH:/gocache" \
    .
  buildah manifest add "${DOCKER_REPO}/kubeletdnat-controller:${PRIMARY_TAG}" "${DOCKER_REPO}/kubeletdnat-controller-${ARCH}:${PRIMARY_TAG}"
done

buildah manifest create "${DOCKER_REPO}/network-interface-manager:${PRIMARY_TAG}"
for ARCH in ${ARCHITECTURES}; do
  echodate "Building network-interface-manager image for $ARCH..."

  buildah bud \
    --tag "${DOCKER_REPO}/network-interface-manager-${ARCH}:${PRIMARY_TAG}" \
    --build-arg "GOPROXY=${GOPROXY:-}" \
    --build-arg "GOCACHE=/gocache" \
    --arch "$ARCH" \
    --override-arch "$ARCH" \
    --format=docker \
    --file cmd/network-interface-manager/Dockerfile.multiarch \
    --volume "$gocaches/$ARCH:/gocache" \
    .
  buildah manifest add "${DOCKER_REPO}/network-interface-manager:${PRIMARY_TAG}" "${DOCKER_REPO}/network-interface-manager-${ARCH}:${PRIMARY_TAG}"
done

rm -rf -- "$gocaches"

# for each given tag, tag and push the image
for TAG in "$@"; do
  if [ -z "$TAG" ]; then
    continue
  fi

  echodate "Tagging as ${TAG}"
  docker tag "${DOCKER_REPO}/kubermatic:${PRIMARY_TAG}" "${DOCKER_REPO}/kubermatic:${TAG}"
  docker tag "${DOCKER_REPO}/nodeport-proxy:${PRIMARY_TAG}" "${DOCKER_REPO}/nodeport-proxy:${TAG}"
  docker tag "${DOCKER_REPO}/addons:${PRIMARY_TAG}" "${DOCKER_REPO}/addons:${TAG}"
  docker tag "${DOCKER_REPO}/etcd-launcher:${PRIMARY_TAG}" "${DOCKER_REPO}/etcd-launcher:${TAG}"
  buildah tag "${DOCKER_REPO}/user-ssh-keys-agent:${PRIMARY_TAG}" "${DOCKER_REPO}/user-ssh-keys-agent:${TAG}"
  buildah tag "${DOCKER_REPO}/kubeletdnat-controller:${PRIMARY_TAG}" "${DOCKER_REPO}/kubeletdnat-controller:${TAG}"
  buildah tag "${DOCKER_REPO}/network-interface-manager:${PRIMARY_TAG}" "${DOCKER_REPO}/network-interface-manager:${TAG}"

  echodate "Pushing images"
  docker push "${DOCKER_REPO}/kubermatic:${TAG}"
  docker push "${DOCKER_REPO}/nodeport-proxy:${TAG}"
  docker push "${DOCKER_REPO}/addons:${TAG}"
  docker push "${DOCKER_REPO}/etcd-launcher:${TAG}"
  buildah manifest push --all "${DOCKER_REPO}/user-ssh-keys-agent:${TAG}" "docker://${DOCKER_REPO}/user-ssh-keys-agent:${TAG}"
  buildah manifest push --all "${DOCKER_REPO}/kubeletdnat-controller:${TAG}" "docker://${DOCKER_REPO}/kubeletdnat-controller:${TAG}"
  buildah manifest push --all "${DOCKER_REPO}/network-interface-manager:${TAG}" "docker://${DOCKER_REPO}/network-interface-manager:${TAG}"
done
