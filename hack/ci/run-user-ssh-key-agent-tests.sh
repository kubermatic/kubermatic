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
export KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ee}"
export ARCHITECTURES="${ARCHITECTURES:-linux/amd64,linux/arm64/v8}"
export TAG_NAME=test

echodate "Building user-ssh-keys-agent..."

start_docker_daemon_ci
docker buildx create --use

# TODO: Do we still need this script and its Prowjob?

docker buildx build \
  --platform "$ARCHITECTURES" \
  --build-arg "GOPROXY=${GOPROXY:-}" \
  --build-arg "KUBERMATIC_EDITION=$KUBERMATIC_EDITION" \
  --file cmd/user-ssh-keys-agent/Dockerfile.multiarch \
  --tag "$DOCKER_REPO/user-ssh-keys-agent:$TAG_NAME" .

echodate "Successfully built for all architectures."
