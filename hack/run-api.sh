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

# Please make sure to set -feature-gates=PrometheusEndpoint=true if you want to use that endpoint.
# Please make sure to set -feature-gates=OIDCKubeCfgEndpoint=true if you want to use that endpoint.

FEATURE_GATES="${FEATURE_GATES:-}"
KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ce}"
KUBERMATIC_DEBUG=${KUBERMATIC_DEBUG:-true}
PPROF_PORT=${PPROF_PORT:-6600}

echodate "Compiling API..."
make kubermatic-api

API_EXTRA_ARGS=""
if [ "$KUBERMATIC_EDITION" == "ee" ]; then
  API_EXTRA_ARGS="-dynamic-datacenters -dynamic-presets"
fi

if [ -z "${VAULT_ADDR:-}" ]; then
  export VAULT_ADDR=https://vault.loodse.com/
fi

if [ -z "${KUBECONFIG:-}" ]; then
  KUBECONFIG=dev.kubeconfig
  vault kv get -field=kubeconfig dev/seed-clusters/dev.kubermatic.io > $KUBECONFIG
fi

SERVICE_ACCOUNT_SIGNING_KEY="${SERVICE_ACCOUNT_SIGNING_KEY:-$(vault kv get -field=service-account-signing-key dev/seed-clusters/dev.kubermatic.io)}"

if [[ "$FEATURE_GATES" =~ "OIDCKubeCfgEndpoint=true" ]]; then
  echodate "Preparing OIDCKubeCfgEndpoint feature..."

  OIDC_ISSUER_CLIENT_ID="${OIDC_ISSUER_CLIENT_ID:-$(vault kv get -field=oidc-issuer-client-id dev/seed-clusters/dev.kubermatic.io)}"
  OIDC_ISSUER_CLIENT_SECRET="${OIDC_ISSUER_CLIENT_SECRET:-$(vault kv get -field=oidc-issuer-client-secret dev/seed-clusters/dev.kubermatic.io)}"
  OIDC_ISSUER_REDIRECT_URI="${OIDC_ISSUER_REDIRECT_URI:-$(vault kv get -field=oidc-issuer-redirect-uri dev/seed-clusters/dev.kubermatic.io)}"
  OIDC_ISSUER_COOKIE_HASH_KEY="${OIDC_ISSUER_COOKIE_HASH_KEY:-$(vault kv get -field=oidc-issuer-cookie-hash-key dev/seed-clusters/dev.kubermatic.io)}"

  API_EXTRA_ARGS="$API_EXTRA_ARGS -oidc-issuer-client-id=$OIDC_ISSUER_CLIENT_ID"
  API_EXTRA_ARGS="$API_EXTRA_ARGS -oidc-issuer-client-secret=$OIDC_ISSUER_CLIENT_SECRET"
  API_EXTRA_ARGS="$API_EXTRA_ARGS -oidc-issuer-redirect-uri=$OIDC_ISSUER_REDIRECT_URI"
  API_EXTRA_ARGS="$API_EXTRA_ARGS -oidc-issuer-cookie-hash-key=$OIDC_ISSUER_COOKIE_HASH_KEY"
fi

echodate "Starting API..."
set -x
./_build/kubermatic-api $API_EXTRA_ARGS \
  -kubeconfig=$KUBECONFIG \
  -ca-bundle=charts/kubermatic-operator/static/ca-bundle.pem \
  -versions=charts/kubermatic/static/master/versions.yaml \
  -updates=charts/kubermatic/static/master/updates.yaml \
  -master-resources=charts/kubermatic/static/master \
  -worker-name="$(worker_name)" \
  -internal-address=127.0.0.1:18085 \
  -prometheus-url=http://localhost:9090 \
  -address=127.0.0.1:8080 \
  -oidc-url=https://dev.kubermatic.io/dex \
  -oidc-authenticator-client-id=kubermatic \
  -service-account-signing-key="$SERVICE_ACCOUNT_SIGNING_KEY" \
  -log-debug=$KUBERMATIC_DEBUG \
  -pprof-listen-address=":$PPROF_PORT" \
  -log-format=Console \
  -logtostderr \
  -v=4 $@
