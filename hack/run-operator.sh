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

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ce}"
NAMESPACE="${NAMESPACE:-}"
KUBERMATIC_DEBUG=${KUBERMATIC_DEBUG:-true}
PPROF_PORT=${PPROF_PORT:-6602}

if [ -z "$NAMESPACE" ]; then
  echo "You must specify a NAMESPACE environment variable to run the operator in."
  echo "Do not use a namespace where another operator is already running."
  echo "Example: NAMESPACE=kubermatic ./hack/run-operator.sh"
  exit 2
fi

if [ -z "${VAULT_ADDR:-}" ]; then
  export VAULT_ADDR=https://vault.kubermatic.com/
fi

if [ -z "${KUBECONFIG:-}" ]; then
  KUBECONFIG=dev.kubeconfig
  vault kv get -field=kubeconfig dev/seed-clusters/dev.kubermatic.io > $KUBECONFIG
fi

export KUBERMATICDOCKERTAG="${KUBERMATICDOCKERTAG:-}"
export UIDOCKERTAG="${UIDOCKERTAG:-}"

if [ -z "$KUBERMATICDOCKERTAG" ]; then
  KUBERMATICDOCKERTAG=$(git rev-parse master)
fi

if [ -z "$UIDOCKERTAG" ]; then
  echo "Finding latest dashboard master revision..."
  UIDOCKERTAG=$(git ls-remote https://github.com/kubermatic/dashboard refs/heads/master | awk '{print $1}')
  echo
fi

echo "Versions"
echo "--------"
echo
echo "  Kubermatic: $KUBERMATICDOCKERTAG (KUBERMATICDOCKERTAG variable)"
echo "  Dashboard : $UIDOCKERTAG (UIDOCKERTAG variable)"
echo

echodate "Compiling operator..."
make kubermatic-operator
echo

echodate "Starting operator..."
set -x
./_build/kubermatic-operator \
  -kubeconfig=$KUBECONFIG \
  -namespace="$NAMESPACE" \
  -worker-name="$(worker_name)" \
  -pprof-listen-address=":${PPROF_PORT}" \
  -log-debug=$KUBERMATIC_DEBUG \
  -log-format=Console \
  -logtostderr \
  -v=4 # Log-level for the Kube dependencies. Increase up to 9 to get request-level logs.
