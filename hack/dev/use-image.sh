#!/usr/bin/env bash

# Copyright 2026 The Kubermatic Kubernetes Platform contributors.
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

# Swaps the kubermatic-operator image on a running local kind environment
# to a locally built tag and waits for the rollout. Because all KKP
# controllers default to the operator's version, one swap updates the
# whole control plane.
#
# Usage (from the repository root):
#   hack/dev/use-image.sh TAG [CLUSTER_NAME] [REPOSITORY]
#
# Examples:
#   hack/dev/use-image.sh dev1
#   hack/dev/use-image.sh dev1 li-smoke
#   hack/dev/use-image.sh dev1 li-smoke localhost:5000/kubermatic
#
# If the tag exists locally as quay.io/kubermatic/kubermatic:TAG (or
# REPOSITORY:TAG), it is loaded into the kind cluster automatically. For
# environments created with --registry, push the image to the registry
# and pass the full REPOSITORY instead.

set -euo pipefail

TAG="${1:?usage: use-image.sh TAG [CLUSTER_NAME] [REPOSITORY]}"
CLUSTER="${2:-kkp-cluster}"
REPOSITORY="${3:-quay.io/kubermatic/kubermatic}"
CONTEXT="kind-${CLUSTER}"
IMAGE="${REPOSITORY}:${TAG}"

if docker image inspect "${IMAGE}" >/dev/null 2>&1; then
  echo "> loading ${IMAGE} into kind cluster ${CLUSTER}"
  kind load docker-image "${IMAGE}" --name "${CLUSTER}"
fi

echo "> upgrading kubermatic-operator to ${IMAGE}"

HELM_ARGS=(--kube-context "${CONTEXT}" --namespace kubermatic --reuse-values --wait
  --set "kubermaticOperator.image.repository=${REPOSITORY}"
  --set "kubermaticOperator.image.tag=${TAG}")

helm upgrade kubermatic-operator charts/kubermatic-operator "${HELM_ARGS[@]}"

kubectl --context "${CONTEXT}" --namespace kubermatic rollout status deploy/kubermatic-operator --timeout=5m

echo "> done; controllers roll to ${TAG} as the operator reconciles them"
