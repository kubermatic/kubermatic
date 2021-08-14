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

### This script tests whether we can successfully build the multiarch
### version for the user-ssh-key-agent.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

export DOCKER_REPO="${DOCKER_REPO:-quay.io/kubermatic}"
export GOOS="${GOOS:-linux}"
export KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ee}"
export ARCHITECTURES=${ARCHITECTURES:-amd64 arm64}
export TAG_NAME=test

# build multi-arch images
buildah manifest create "${DOCKER_REPO}/user-ssh-keys-agent:${TAG_NAME}"
for ARCH in ${ARCHITECTURES}; do
  # Building via buildah does not use the gocache, but that's okay, because we
  # wouldn't want to cache arm64 stuff anyway, as it would just blow up the
  # cache size and force every e2e test to download gigabytes worth of unneeded
  # arm64 stuff. We might need to change this once we run e2e tests on arm64.
  buildah bud \
    --tag "${DOCKER_REPO}/user-ssh-keys-agent-${ARCH}:${TAG_NAME}" \
    --arch "$ARCH" \
    --override-arch "$ARCH" \
    --format=docker \
    --file cmd/user-ssh-keys-agent/Dockerfile.multiarch \
    .
  buildah manifest add "${DOCKER_REPO}/user-ssh-keys-agent:${TAG_NAME}" "${DOCKER_REPO}/user-ssh-keys-agent-${ARCH}:${TAG_NAME}"
done
