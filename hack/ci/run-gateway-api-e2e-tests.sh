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
### from the start (fresh install), deploys a test cluster, and then runs the
### Gateway API e2e tests.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

export KIND_CLUSTER_NAME="${SEED_NAME:-kubermatic}"
export KUBERMATIC_YAML=hack/ci/testdata/kubermatic.yaml

# Create a patched config with HeadlessInstallation disabled for Gateway API tests.
# Envoy Gateway deployment is skipped when HeadlessInstallation is true, but we need
# it deployed for Gateway API e2e tests. This creates a temp config with the flag
# disabled, avoiding modification of the shared base config file.
GATEWAY_KUBERMATIC_YAML="$(mktemp)"
cp "$KUBERMATIC_YAML" "$GATEWAY_KUBERMATIC_YAML"
sed -i "s/HeadlessInstallation: true/HeadlessInstallation: false/g" "$GATEWAY_KUBERMATIC_YAML"
# Add auth section with dummy serviceAccountKey (required when HeadlessInstallation=false)
# Tests don't use authentication, but validation requires this field.
# Note: Using sed heredoc for Linux/GNU sed compatibility in CI.
# yq was causing helm template failures due to YAML processing differences.
sed -i "/^spec:/a\\
  auth:\\
    serviceAccountKey: \"KiehK9IqLofm4_lJDPwlsa-hwEqe4H74X9OO_dbTYQs\"
" "$GATEWAY_KUBERMATIC_YAML"
export KUBERMATIC_YAML="$GATEWAY_KUBERMATIC_YAML"

# Enable Gateway API mode for fresh installation
# Skip Dex deployment as tests don't use authentication
export INSTALLER_FLAGS="--migrate-gateway-api --skip-charts=dex"

source hack/ci/setup-kind-cluster.sh

# gather the logs of all things in the cluster control plane and in the Kubermatic namespace
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/cluster-control-plane" --namespace 'cluster-*' > /dev/null 2>&1 &
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/envoy-gateway" --namespace envoy-gateway-controller > /dev/null 2>&1 &

source hack/ci/setup-kubermatic-in-kind.sh

export GIT_HEAD_HASH="$(git rev-parse HEAD | tr -d '\n')"

if [ -z "${E2E_SSH_PUBKEY:-}" ]; then
  echodate "Getting default SSH pubkey for machines from Vault"
  retry 5 vault_ci_login
  E2E_SSH_PUBKEY="$(mktemp)"
  vault kv get -field=pubkey dev/e2e-machine-controller-ssh-key > "${E2E_SSH_PUBKEY}"
else
  E2E_SSH_PUBKEY_CONTENT="${E2E_SSH_PUBKEY}"
  E2E_SSH_PUBKEY="$(mktemp)"
  echo "${E2E_SSH_PUBKEY_CONTENT}" > "${E2E_SSH_PUBKEY}"
fi

echodate "SSH public key will be $(head -c 25 ${E2E_SSH_PUBKEY})...$(tail -c 25 ${E2E_SSH_PUBKEY})"

# Verify Gateway API resources exist before running tests
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
  -test.run "TestGatewayAPIFreshInstall|TestGatewayAPINamespaceLabel" \
  -aws-kkp-datacenter "$AWS_E2E_TESTS_DATACENTER" \
  -ssh-pub-key "$(cat "$E2E_SSH_PUBKEY")"

echodate "Gateway API fresh install tests completed successfully!"
