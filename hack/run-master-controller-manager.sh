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
KUBERMATIC_DEBUG=${KUBERMATIC_DEBUG:-true}
PPROF_PORT=${PPROF_PORT:-6601}

echodate "Compiling master-controller-manager..."
make master-controller-manager

CTRL_EXTRA_ARGS=""
if [ "$KUBERMATIC_EDITION" == "ee" ]; then
  CTRL_EXTRA_ARGS="-dynamic-datacenters"
fi

if [ -z "${VAULT_ADDR:-}" ]; then
  export VAULT_ADDR=https://vault.loodse.com/
fi

if [ -z "${KUBECONFIG:-}" ]; then
  KUBECONFIG=dev.kubeconfig
  vault kv get -field=kubeconfig dev/seed-clusters/dev.kubermatic.io > $KUBECONFIG
fi

echodate "Starting master-controller-manager..."
set -x
./_build/master-controller-manager $CTRL_EXTRA_ARGS \
  -kubeconfig=$KUBECONFIG \
  -enable-leader-election=false \
  -internal-address=127.0.0.1:8086 \
  -worker-name="$(worker_name)" \
  -log-debug=$KUBERMATIC_DEBUG \
  -pprof-listen-address=":${PPROF_PORT}" \
  -log-format=Console \
  -logtostderr \
  -v=4 # Log-level for the Kube dependencies. Increase up to 9 to get request-level logs.
