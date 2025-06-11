#!/usr/bin/env bash

# Copyright 2023 The Kubermatic Kubernetes Platform contributors.
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

cd $(dirname $0)/../..
source hack/lib.sh

export GOCACHE_BASE=$(mktemp -d)

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"
TARGET_DIRECTORY=$GOCACHE_BASE/amd64 make download-gocache

trivy() {
  mkdir -p /tmp/trivy-cache
  time docker run \
    --rm \
    -v "$(realpath .):/opt/src" \
    -v /run/docker.sock:/var/run/docker.sock \
    -v /tmp/trivy-cache:/cache \
    -e "TRIVY_DB_REPOSITORY=public.ecr.aws/aquasecurity/trivy-db" \
    -e "TRIVY_JAVA_DB_REPOSITORY=public.ecr.aws/aquasecurity/trivy-java-db" \
    -w /opt/src \
    aquasec/trivy:0.56.2 --cache-dir /cache "$@"
}

export KUBERMATIC_VERSION="${KUBERMATIC_VERSION:-$(git rev-parse HEAD)}"
export DOCKER_REPO=quay.io/kubermatic
export TAG=local
export ARCHITECTURES=amd64 # save time by skipping building/scanning arm64
export UIDOCKERTAG=NA      # prevent the need for an SSH key to git ls-remote the dashboard version

REPOSUFFIX=""
if [ "$KUBERMATIC_EDITION" != "ce" ]; then
  REPOSUFFIX="-$KUBERMATIC_EDITION"
fi

images=(
  addons
  etcd-launcher
  kubeletdnat-controller
  kubermatic
  network-interface-manager
  nodeport-proxy
  user-ssh-keys-agent
)

TEST_NAME="Create container images"
echodate "Creating container images..."
hack/images/release.sh "${images[@]}"

echodate "Images built successfully."

TEST_NAME="Scan Go dependencies"
echodate "Scanning Go dependencies..."
trivy fs --quiet .

# improve readability from far away
echo
echo
echo

for image in "${images[@]}"; do
  TEST_NAME="Scan $image image"
  echodate "Scanning $image image..."

  if [[ $image == kubermatic ]]; then
    image="$image$REPOSUFFIX"
  fi

  trivy image --quiet "$DOCKER_REPO/$image:$TAG"

  # improve readability from far away
  echo
  echo
  echo
done

echodate "Scans completed."
