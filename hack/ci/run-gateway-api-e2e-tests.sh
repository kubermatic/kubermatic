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

### This script sets up a local KKP installation in kind with Gateway API enabled
### from the start (fresh install) and runs the Gateway API e2e tests.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

export KIND_CLUSTER_NAME="${SEED_NAME:-kubermatic}"
export KUBERMATIC_YAML=hack/ci/testdata/kubermatic_gatewayapi.yaml

echodate "Deploying KKP with Envoy Gateway"

source hack/ci/setup-kind-cluster.sh

# gather the logs of all things in the cluster control plane and in the Kubermatic namespace
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/cluster-control-plane" --namespace 'cluster-*' > /dev/null 2>&1 &
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/envoy-gateway" --namespace envoy-gateway-controller > /dev/null 2>&1 &

KUBERMATIC_VERSION="${KUBERMATIC_VERSION:-$(git rev-parse HEAD)}"

DEX_PASSWORD_HASH='$2y$10$Lurps56wlfD5Rgelz9u4FuYOMdUw8FZaIKyt5xUyPBwHP0Eo.yLhW'

export HELM_VALUES_EXTRA="
migrateGatewayAPI: true
dex:
  ingress:
    enabled: false
    hosts: []
    tls: []
  config:
    issuer: https://ci.kubermatic.io/dex
    enablePasswordDB: true
    staticPasswords:
      - email: kubermatic@example.com
        hash: \"${DEX_PASSWORD_HASH}\"
        username: admin
httproute:
  gatewayName: kubermatic
  gatewayNamespace: kubermatic
  domain: ci.kubermatic.io
  timeout: 3600s
# if we deploy envoy proxy as LB, its status won't be happy until an external LB IP is assigned
# which does not happen in kind without extra tooling/setup. Therefore, we deploy it as NodePort for now...
envoyProxy:
  service:
    type: NodePort
    externalTrafficPolicy: Cluster
"

export INSTALLER_FLAGS="--migrate-gateway-api"
source hack/ci/setup-kubermatic-in-kind.sh

echodate "Verifying Gateway API resources are deployed..."
echodate "Checking Gateway resource kubermatic/kubermatic"
retry 10 kubectl get gateway -n kubermatic kubermatic
echodate "Checking HTTPRoute resource kubermatic/kubermatic"
retry 10 kubectl get httproute -n kubermatic kubermatic
echodate "Checking GatewayClass resource kubermatic-envoy"
retry 10 kubectl get gatewayclass kubermatic-envoy

echodate "Gateway API resources are present."
echodate "Running Gateway API fresh install tests..."

go_test gateway_api_e2e -timeout 1h -tags e2e -v ./pkg/test/e2e/gateway-api \
  -test.run "TestGatewayAPIFreshInstall"

echodate "Gateway API fresh install tests completed successfully!"
