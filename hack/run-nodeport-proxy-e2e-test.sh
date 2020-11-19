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

### This script sets up a local KKP installation in kind, deploys a
### couple of test Presets and Users and then runs the e2e tests for the
### API.

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

function clean-up {
    echodate "Deleting cluster ${KIND_CLUSTER_NAME}"
    kind delete cluster --name "${KIND_CLUSTER_NAME}" || true
}
trap clean-up EXIT

DOCKER_REPO="${DOCKER_REPO:-quay.io/kubermatic}"
GOOS="${GOOS:-linux}"
TAG="$(git rev-parse HEAD)"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kubermatic}"
KIND_NODE_VERSION="${KIND_NODE_VERSION:-v1.18.2}"

# build Docker images
make -C cmd/nodeport-proxy docker \
    GOOS="${GOOS}" \
    DOCKER_REPO="${DOCKER_REPO}" \
    TAG="${TAG}"

# setup Kind cluster
time retry 5 kind create cluster --name "${KIND_CLUSTER_NAME}" --image=kindest/node:"${KIND_NODE_VERSION}"
# load nodeport-proxy image
time retry 5 kind load docker-image "${DOCKER_REPO}/nodeport-proxy:${TAG}" --name "$KIND_CLUSTER_NAME"

# Get ginkgo
go get -v github.com/onsi/ginkgo/ginkgo
# run tests
ginkgo -v pkg/test/e2e/nodeport-proxy/ \
    -r randomizeAllSpecs \
    --randomizeSuites \
    --failOnPending \
    --cover \
    --trace \
    --race \
    --progress \
    -- --kubeconfig "${HOME}/.kube/config" \
    --kubermatic-tag "${TAG}" \
    --skip-cleanup \
