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

### This script tests the migration from nginx-ingress-controller to Gateway API.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

export KIND_CLUSTER_NAME="${SEED_NAME:-kubermatic}"
export KUBERMATIC_YAML=hack/ci/testdata/kubermatic_nginx.yaml

echodate "Deploying KKP with nginx-ingress"

source hack/ci/setup-kind-cluster.sh

# gather the logs of all things in the cluster control plane and in the Kubermatic namespace
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/cluster-control-plane" --namespace 'cluster-*' > /dev/null 2>&1 &
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/nginx-ingress" --namespace nginx-ingress-controller > /dev/null 2>&1 &

KUBERMATIC_VERSION="${KUBERMATIC_VERSION:-$(git rev-parse HEAD)}"

DEX_PASSWORD_HASH='$2y$10$Lurps56wlfD5Rgelz9u4FuYOMdUw8FZaIKyt5xUyPBwHP0Eo.yLhW'

export HELM_VALUES_EXTRA="
dex:
  replicaCount: 1
  ingress:
    enabled: true
    className: "nginx"
    hosts:
      - host: ci.kubermatic.io
        paths:
          - path: /dex
            pathType: ImplementationSpecific
    tls: []
    annotations:
      cert-manager.io/cluster-issuer: letsencrypt-staging

  config:
    issuer: https://ci.kubermatic.io/dex
    enablePasswordDB: true
    staticPasswords:
      - email: kubermatic@example.com
        hash: ${DEX_PASSWORD_HASH}
        username: admin
"

source hack/ci/setup-kubermatic-in-kind.sh

retry 10 check_all_deployments_ready nginx-ingress-controller
echodate "nginx-ingress controller deployed"

echodate "Running pre-migration tests (Ingress mode)..."
go_test gateway_api_migration_e2e -timeout 1h -tags e2e -v ./pkg/test/e2e/gateway-api -test.run "TestGatewayAPIPreMigration"

echodate "Pre-migration tests passed"
echodate ""
echodate "Upgrading to Gateway API mode"

export HELM_VALUES_EXTRA="
migrateGatewayAPI: true
dex:
  replicaCount: 1
  ingress:
    enabled: false
    hosts: []
    tls: []
  config:
    issuer: https://ci.kubermatic.io/dex
    enablePasswordDB: true
    staticPasswords:
      - email: kubermatic@example.com
        hash: ${DEX_PASSWORD_HASH}
        username: admin
httpRoute:
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

merged_helm_values_file="$(mktemp)"
echo "$HELM_VALUES_STR" >> $merged_helm_values_file
yq e 'del(.dex)' -i $merged_helm_values_file
echo "$HELM_VALUES_EXTRA" >> $merged_helm_values_file

echodate "Re-running kubermatic-installer with --migrate-gateway-api flag..."

./_build/kubermatic-installer deploy kubermatic-master \
  --storageclass copy-default \
  --config "$KUBERMATIC_CONFIG" \
  --helm-values "$merged_helm_values_file" \
  --skip-seed-validation=kubermatic \
  --migrate-gateway-api \
  --verbose

protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/envoy-gateway" --namespace envoy-gateway-controller > /dev/null 2>&1 &

echodate "Waiting for Kubermatic Operator to restart with Gateway API enabled..."
sleep 5
retry 10 check_all_deployments_ready kubermatic
echodate "Operator restarted with Gateway API mode"


echodate "Verifying Gateway API resources deployed..."
retry 10 check_all_deployments_ready envoy-gateway-controller
retry 10 check_all_deployments_ready kubermatic

echodate "Running post-migration tests (Gateway API mode)..."

go_test gateway_api_migration_e2e -timeout 1h -tags e2e -v ./pkg/test/e2e/gateway-api -test.run "TestGatewayAPIPostMigration"

echodate "Gateway API migration tests completed successfully!"
