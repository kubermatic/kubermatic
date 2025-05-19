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
KUBERMATIC_WORKERNAME=${KUBERMATIC_WORKERNAME:-$(uname -n)}
KUBERMATIC_DEBUG=${KUBERMATIC_DEBUG:-true}
KUBERMATIC_SEED=${KUBERMATIC_SEED:-hamburg}
KUBERMATIC_EXTERNAL_URL=${KUBERMATIC_EXTERNAL_URL:-dev.kubermatic.io}
PPROF_PORT=${PPROF_PORT:-6600}

# Deploy a user-cluster/ipam-controller for which we actually
# have a pushed image
echodate "Compiling seed-controller-manager..."
export KUBERMATICCOMMIT="${KUBERMATICCOMMIT:-$(git rev-parse origin/main)}"
make seed-controller-manager

CTRL_EXTRA_ARGS=""

if [ -n "${CONFIG_FILE:-}" ]; then
  CTRL_EXTRA_ARGS="-kubermatic-configuration-file=$CONFIG_FILE"
fi

if [ -z "${VAULT_ADDR:-}" ]; then
  export VAULT_ADDR=https://vault.kubermatic.com/
fi

if [ -z "${KUBECONFIG:-}" ]; then
  KUBECONFIG=dev.kubeconfig
  vault kv get -field=kubeconfig dev/seed-clusters/$KUBERMATIC_EXTERNAL_URL > $KUBECONFIG
fi

if [ -z "${DOCKERCONFIGJSON:-}" ]; then
  DOCKERCONFIGJSON=dev.dockerconfigjson
  vault kv get -field=.dockerconfigjson dev/seed-clusters/dev.kubermatic.io > $DOCKERCONFIGJSON
fi

OIDC_ISSUER_URL="${OIDC_ISSUER_URL:-$(vault kv get -field=oidc-issuer-url dev/seed-clusters/dev.kubermatic.io)}"
OIDC_ISSUER_CLIENT_ID="${OIDC_ISSUER_CLIENT_ID:-$(vault kv get -field=oidc-issuer-client-id dev/seed-clusters/dev.kubermatic.io)}"
OIDC_ISSUER_CLIENT_SECRET="${OIDC_ISSUER_CLIENT_SECRET:-$(vault kv get -field=oidc-issuer-client-secret dev/seed-clusters/dev.kubermatic.io)}"

if [ -z "${CA_BUNDLE:-}" ]; then
  CA_BUNDLE=charts/kubermatic-operator/static/ca-bundle.pem
fi

echodate "Starting seed-controller-manager..."
set -x
./_build/seed-controller-manager $CTRL_EXTRA_ARGS \
  -namespace=kubermatic \
  -enable-leader-election=false \
  -seed-name=dev \
  -kubeconfig=$KUBECONFIG \
  -ca-bundle=$CA_BUNDLE \
  -addons-path=addons \
  -feature-gates=OpenIDAuthPlugin=true \
  -worker-name="$(worker_name)" \
  -external-url=$KUBERMATIC_EXTERNAL_URL \
  -docker-pull-config-json-file=$DOCKERCONFIGJSON \
  -oidc-issuer-url=$OIDC_ISSUER_URL \
  -oidc-issuer-client-id=$OIDC_ISSUER_CLIENT_ID \
  -oidc-issuer-client-secret=$OIDC_ISSUER_CLIENT_SECRET \
  -log-debug=$KUBERMATIC_DEBUG \
  -log-format=Console \
  -max-parallel-reconcile=10 \
  -pprof-listen-address=":${PPROF_PORT}" \
  -logtostderr \
  -v=4 # Log-level for the Kube dependencies. Increase up to 9 to get request-level logs.
