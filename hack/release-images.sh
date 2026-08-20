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

### Builds and pushes all KKP container images:
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
### `./release-images.sh foo bar` will tag `kubermatic:foo` and
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

mkdir -p _dist/images

export ALL_TAGS=$@
export DOCKER_REPO="${DOCKER_REPO:-quay.io/kubermatic}"
export GOOS="${GOOS:-linux}"
export KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ee}"
export ARCHITECTURES="${ARCHITECTURES:-amd64 arm64}"

REPOSUFFIX=""
if [ "$KUBERMATIC_EDITION" != "ce" ]; then
  REPOSUFFIX="-$KUBERMATIC_EDITION"
fi

generate_image_sbom() {
  local imageRef="$1"
  local outFile="$2"

  echodate "Generating SBOM for $imageRef..."
  syft "docker:$imageRef" -o "spdx-json=$outFile"
}

declare -A ATTACHED_REFS

attach_image_sbom() {
  local imageRef="$1"
  local sbomFile="$2"

  local repo="${imageRef%:*}"
  local digest
  digest="$(oras manifest fetch --descriptor "$imageRef" | jq -er '.digest')"
  local ref="$repo@$digest"
  local key="$ref:$(basename "$sbomFile")"

  if [[ -n "${ATTACHED_REFS[$key]:-}" ]]; then
    echodate "SBOM $(basename "$sbomFile") already attached to $ref, skipping duplicate for $imageRef..."
    return
  fi

  echodate "Attaching SBOM $(basename "$sbomFile") to $ref..."
  oras attach --artifact-type application/spdx+json \
    "$ref" "$sbomFile:application/spdx+json"

  ATTACHED_REFS[$key]=1
}

# build Docker images
PRIMARY_TAG="${1}"
VERSION_LABEL="org.opencontainers.image.version=${KUBERMATICDOCKERTAG:-$PRIMARY_TAG}"
make docker-build TAGS="$PRIMARY_TAG"
make -C cmd/nodeport-proxy docker TAG="$PRIMARY_TAG"
docker build --label "$VERSION_LABEL" -t "$DOCKER_REPO/addons:$PRIMARY_TAG" addons
docker build --label "$VERSION_LABEL" -t "$DOCKER_REPO/etcd-launcher:$PRIMARY_TAG" -f cmd/etcd-launcher/Dockerfile .
docker build --label "$VERSION_LABEL" -t "$DOCKER_REPO/conformance-tests:$PRIMARY_TAG" -f cmd/conformance-tester/Dockerfile .

generate_image_sbom "$DOCKER_REPO/kubermatic$REPOSUFFIX:$PRIMARY_TAG" "_dist/images/kubermatic$REPOSUFFIX.sbom.spdx.json"
generate_image_sbom "$DOCKER_REPO/nodeport-proxy:$PRIMARY_TAG" "_dist/images/nodeport-proxy.sbom.spdx.json"
generate_image_sbom "$DOCKER_REPO/addons:$PRIMARY_TAG" "_dist/images/addons.sbom.spdx.json"
generate_image_sbom "$DOCKER_REPO/etcd-launcher:$PRIMARY_TAG" "_dist/images/etcd-launcher.sbom.spdx.json"
generate_image_sbom "$DOCKER_REPO/conformance-tests:$PRIMARY_TAG" "_dist/images/conformance-tests.sbom.spdx.json"

# switch to a multi platform-enabled builder
docker buildx create --use

# get gocache in all archs
GOCACHE_BASE="$(mktemp -d)"

for arch in $ARCHITECTURES; do
  echodate "Downloading gocache for $arch…"
  TARGET_DIRECTORY="$GOCACHE_BASE/$arch" GOARCH="$arch" ./hack/ci/download-gocache.sh
done

buildx_build() {
  local context="$1"
  local file="$2"
  local repository="$3"
  local tag="$4"

  for arch in $ARCHITECTURES; do
    # move relevant gocache temporarily here
    mv "$GOCACHE_BASE/$arch" .gocache

    # --load requires to build architectures individually, only
    # --push would support building multiple archs at the same time.
    docker buildx build \
      --load \
      --platform "linux/$arch" \
      --build-arg "GOPROXY=${GOPROXY:-}" \
      --build-arg "GOCACHE=/go/src/k8c.io/kubermatic/.gocache" \
      --build-arg "KUBERMATIC_EDITION=$KUBERMATIC_EDITION" \
      --provenance false \
      --file "$file" \
      --tag "$repository:$tag-$arch" \
      --label "$VERSION_LABEL" \
      $context

    # move gocache back
    mv .gocache "$GOCACHE_BASE/$arch"
  done
}

create_manifest() {
  local repository="$1"
  local builtTag="$2"
  local publishedTag="$3"

  # push all images that are involved in this new manifest
  images=""
  for arch in $ARCHITECTURES; do
    publishedName="$repository:$publishedTag-$arch"
    images="$images $publishedName"

    docker tag "$repository:$builtTag-$arch" "$publishedName"
    docker push "$publishedName"
  done

  docker manifest create --amend "$repository:$publishedTag" $images

  for arch in $ARCHITECTURES; do
    docker manifest annotate --arch "$arch" "$repository:$publishedTag" "$repository:$publishedTag-$arch"
  done
}

# build multi-arch images
echodate "Building user-ssh-keys-agent images..."
buildx_build . cmd/user-ssh-keys-agent/Dockerfile.multiarch "$DOCKER_REPO/user-ssh-keys-agent" "$PRIMARY_TAG"

for arch in $ARCHITECTURES; do
  generate_image_sbom "$DOCKER_REPO/user-ssh-keys-agent:$PRIMARY_TAG-$arch" "_dist/images/user-ssh-keys-agent-$arch.sbom.spdx.json"
done

echodate "Building kubeletdnat-controller images..."
buildx_build . cmd/kubeletdnat-controller/Dockerfile.multiarch "$DOCKER_REPO/kubeletdnat-controller" "$PRIMARY_TAG"

for arch in $ARCHITECTURES; do
  generate_image_sbom "$DOCKER_REPO/kubeletdnat-controller:$PRIMARY_TAG-$arch" "_dist/images/kubeletdnat-controller-$arch.sbom.spdx.json"
done

echodate "Building network-interface-manager images..."
buildx_build . cmd/network-interface-manager/Dockerfile.multiarch "$DOCKER_REPO/network-interface-manager" "$PRIMARY_TAG"

for arch in $ARCHITECTURES; do
  generate_image_sbom "$DOCKER_REPO/network-interface-manager:$PRIMARY_TAG-$arch" "_dist/images/network-interface-manager-$arch.sbom.spdx.json"
done

# for each given tag, tag and push the image
for TAG in $ALL_TAGS; do
  if [ -z "$TAG" ]; then
    continue
  fi

  echodate "Tagging as $TAG"
  docker tag "$DOCKER_REPO/kubermatic$REPOSUFFIX:$PRIMARY_TAG" "$DOCKER_REPO/kubermatic$REPOSUFFIX:$TAG"
  docker tag "$DOCKER_REPO/nodeport-proxy:$PRIMARY_TAG" "$DOCKER_REPO/nodeport-proxy:$TAG"
  docker tag "$DOCKER_REPO/addons:$PRIMARY_TAG" "$DOCKER_REPO/addons:$TAG"
  docker tag "$DOCKER_REPO/etcd-launcher:$PRIMARY_TAG" "$DOCKER_REPO/etcd-launcher:$TAG"
  docker tag "$DOCKER_REPO/conformance-tests:$PRIMARY_TAG" "$DOCKER_REPO/conformance-tests:$TAG"

  if [ -z "${NO_PUSH:-}" ]; then
    echodate "Pushing images"

    docker push "$DOCKER_REPO/kubermatic$REPOSUFFIX:$TAG"
    attach_image_sbom "$DOCKER_REPO/kubermatic$REPOSUFFIX:$TAG" "_dist/images/kubermatic$REPOSUFFIX.sbom.spdx.json"

    docker push "$DOCKER_REPO/nodeport-proxy:$TAG"
    attach_image_sbom "$DOCKER_REPO/nodeport-proxy:$TAG" "_dist/images/nodeport-proxy.sbom.spdx.json"

    docker push "$DOCKER_REPO/addons:$TAG"
    attach_image_sbom "$DOCKER_REPO/addons:$TAG" "_dist/images/addons.sbom.spdx.json"

    docker push "$DOCKER_REPO/etcd-launcher:$TAG"
    attach_image_sbom "$DOCKER_REPO/etcd-launcher:$TAG" "_dist/images/etcd-launcher.sbom.spdx.json"

    docker push "$DOCKER_REPO/conformance-tests:$TAG"
    attach_image_sbom "$DOCKER_REPO/conformance-tests:$TAG" "_dist/images/conformance-tests.sbom.spdx.json"

    create_manifest "$DOCKER_REPO/user-ssh-keys-agent" "$PRIMARY_TAG" "$TAG"
    create_manifest "$DOCKER_REPO/kubeletdnat-controller" "$PRIMARY_TAG" "$TAG"
    create_manifest "$DOCKER_REPO/network-interface-manager" "$PRIMARY_TAG" "$TAG"

    docker manifest push --purge "$DOCKER_REPO/user-ssh-keys-agent:$TAG"
    for arch in $ARCHITECTURES; do
      attach_image_sbom "$DOCKER_REPO/user-ssh-keys-agent:$TAG-$arch" "_dist/images/user-ssh-keys-agent-$arch.sbom.spdx.json"
    done

    docker manifest push --purge "$DOCKER_REPO/kubeletdnat-controller:$TAG"
    for arch in $ARCHITECTURES; do
      attach_image_sbom "$DOCKER_REPO/kubeletdnat-controller:$TAG-$arch" "_dist/images/kubeletdnat-controller-$arch.sbom.spdx.json"
    done

    docker manifest push --purge "$DOCKER_REPO/network-interface-manager:$TAG"
    for arch in $ARCHITECTURES; do
      attach_image_sbom "$DOCKER_REPO/network-interface-manager:$TAG-$arch" "_dist/images/network-interface-manager-$arch.sbom.spdx.json"
    done
  fi
done

if [ -n "${NO_PUSH:-}" ]; then
  echodate "Not pushing images because \$NO_PUSH is set."
fi
