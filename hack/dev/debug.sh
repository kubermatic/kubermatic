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

# Runs a KKP component locally on this machine against a kind
# environment instead of inside the cluster: scales the in-cluster
# Deployment to 0, mirrors its container args into a `go run` call and
# restores the Deployment on exit. Restarting is then a matter of
# re-running this script, and breakpoints/delve work as usual.
#
# Usage (from the repository root):
#   hack/dev/debug.sh COMPONENT [CLUSTER_NAME]
#
# Components: master-controller-manager, seed-controller-manager,
#             kubermatic-webhook
# (user-cluster-controller-manager is per-cluster and not supported yet.)
#
# The in-cluster args are copied from the live Deployment so flag drift
# in the operator does not invalidate this script; leader election is
# disabled for the local run.

set -euo pipefail

COMPONENT="${1:?usage: debug.sh COMPONENT [CLUSTER_NAME]}"
CLUSTER="${2:-kkp-cluster}"
CONTEXT="kind-${CLUSTER}"

case "${COMPONENT}" in
  master-controller-manager|seed-controller-manager|kubermatic-webhook)
    ;;
  user-cluster-controller-manager)
    echo "unsupported: user-cluster-controller-manager runs per user cluster; debug it via its cluster namespace" >&2
    exit 1
    ;;
  *)
    echo "unknown component '${COMPONENT}', expected master-controller-manager, seed-controller-manager or kubermatic-webhook" >&2
    exit 1
    ;;
esac

DEPLOYMENT="kubermatic-${COMPONENT}"
NAMESPACE="kubermatic"

echo "> scaling ${NAMESPACE}/${DEPLOYMENT} to 0"
kubectl --context "${CONTEXT}" --namespace "${NAMESPACE}" scale deploy "${DEPLOYMENT}" --replicas=0

restore() {
  echo
  echo "> restoring: kubectl --context ${CONTEXT} -n ${NAMESPACE} scale deploy ${DEPLOYMENT} --replicas=1"
  kubectl --context "${CONTEXT}" --namespace "${NAMESPACE}" scale deploy "${DEPLOYMENT}" --replicas=1 || true
}
trap restore EXIT

# mirror the args the operator gave the in-cluster instance, minus leader election
ARGS="$(kubectl --context "${CONTEXT}" --namespace "${NAMESPACE}" get deploy "${DEPLOYMENT}" \
  -o jsonpath='{.spec.template.spec.containers[0].args}' \
  | tr ' ' '\n' | grep -v -- '--enable-leader-election' | tr '\n' ' ')"

echo "> running ${COMPONENT} locally (Ctrl-C to stop; the Deployment is restored on exit)"
echo "> go run ./cmd/${COMPONENT} ${ARGS} --enable-leader-election=false"

# shellcheck disable=SC2086
exec env KUBECONFIG="${KUBECONFIG:-$HOME/.kube/config}" \
  go run "./cmd/${COMPONENT}" ${ARGS} --enable-leader-election=false
