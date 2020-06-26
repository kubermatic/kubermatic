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

set -exuo pipefail

make -C $(dirname $0)/.. kubermatic-api

KUBERMATIC_WORKERNAME=${KUBERMATIC_WORKERNAME:-$(uname -n)}
KUBERMATIC_DEBUG=${KUBERMATIC_DEBUG:-true}
PPROF_PORT=${PPROF_PORT:-6600}

# Please make sure to set -feature-gates=PrometheusEndpoint=true if you want to use that endpoint.

# Please make sure to set -feature-gates=OIDCKubeCfgEndpoint=true if you want to use that endpoint.
# Note that you would have to pass a few additional flags as well.

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
./_build/kubermatic-api \
  -kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -dynamic-datacenters=true \
  -dynamic-presets=true \
  -versions=../config/kubermatic/static/master/versions.yaml \
  -updates=../config/kubermatic/static/master/updates.yaml \
  -master-resources=../config/kubermatic/static/master \
  -worker-name="$(tr -cd '[:alnum:]' <<< $KUBERMATIC_WORKERNAME | tr '[:upper:]' '[:lower:]')" \
  -internal-address=127.0.0.1:18085 \
  -prometheus-url=http://localhost:9090 \
  -address=127.0.0.1:8080 \
  -oidc-url=https://dev.kubermatic.io/dex \
  -oidc-authenticator-client-id=kubermatic \
  -oidc-issuer-client-id="$(vault kv get -field=oidc-issuer-client-id dev/seed-clusters/dev.kubermatic.io)" \
  -oidc-issuer-client-secret="$(vault kv get -field=oidc-issuer-client-secret dev/seed-clusters/dev.kubermatic.io)" \
  -oidc-issuer-redirect-uri="$(vault kv get -field=oidc-issuer-redirect-uri dev/seed-clusters/dev.kubermatic.io)" \
  -oidc-issuer-cookie-hash-key="$(vault kv get -field=oidc-issuer-cookie-hash-key dev/seed-clusters/dev.kubermatic.io)" \
  -service-account-signing-key="$(vault kv get -field=service-account-signing-key dev/seed-clusters/dev.kubermatic.io)" \
  -log-debug=$KUBERMATIC_DEBUG \
  -pprof-listen-address=":${PPROF_PORT}" \
  -log-format=Console \
  -logtostderr \
  -v=4 $@
