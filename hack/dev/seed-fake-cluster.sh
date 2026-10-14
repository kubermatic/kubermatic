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

# Creates a minimal fake Cluster plus one Addon in its namespace, the
# proven input shape for exercising seed-side collectors and controllers
# without provisioning a real user cluster (validated by the dogfood run
# 2026-09-19: controllers skip Clusters without status; Addons need an
# explicit spec.name and spec.cluster.name).
#
# Usage (from the repository root):
#   hack/dev/seed-fake-cluster.sh NAME [CLUSTER_NAME]
#
# Example:
#   hack/dev/seed-fake-cluster.sh acs-prod li6-kkp

set -euo pipefail

NAME="${1:?usage: seed-fake-cluster.sh NAME [KIND_CLUSTER_NAME]}"
CLUSTER="${2:-kkp-cluster}"
CONTEXT="kind-${CLUSTER}"
NAMESPACE="cluster-${NAME}"

kubectl --context "${CONTEXT}" create namespace "${NAMESPACE}" --dry-run=client -o yaml |
  kubectl --context "${CONTEXT}" apply -f -

kubectl --context "${CONTEXT}" apply -f - << EOF
apiVersion: kubermatic.k8c.io/v1
kind: Cluster
metadata:
  name: ${NAME}
---
apiVersion: kubermatic.k8c.io/v1
kind: Addon
metadata:
  name: csi
  namespace: ${NAMESPACE}
spec:
  name: csi
  cluster:
    name: ${NAME}
EOF

echo "> seeded Cluster ${NAME} + Addon csi in ${NAMESPACE}; metrics label: cluster=\"${NAME}\""
