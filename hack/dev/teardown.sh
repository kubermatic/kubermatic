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

# Tears down a local kind environment: the kind cluster and, optionally,
# the registry container that served it. Also removes build outputs.
#
# Usage (from the repository root):
#   hack/dev/teardown.sh KIND_CLUSTER_NAME [REGISTRY_CONTAINER]
#
# Example:
#   hack/dev/teardown.sh my-kkp my-registry

set -euo pipefail

CLUSTER="${1:?usage: teardown.sh KIND_CLUSTER_NAME [REGISTRY_CONTAINER]}"
REGISTRY="${2:-}"

kind delete cluster --name "${CLUSTER}"

if [ -n "${REGISTRY}" ]; then
  docker rm -f "${REGISTRY}" || true
fi

rm -rf _build

echo "> torn down: kind cluster ${CLUSTER}${REGISTRY:+, registry ${REGISTRY}}, _build/"
