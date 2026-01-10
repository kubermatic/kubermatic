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
export KUBERMATIC_YAML=hack/ci/testdata/kubermatic.yaml

echodate "Deploying KKP with nginx-ingress"

export INSTALLER_FLAGS=""

source hack/ci/setup-kind-cluster.sh

# gather the logs of all things in the cluster control plane and in the Kubermatic namespace
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/cluster-control-plane" --namespace 'cluster-*' > /dev/null 2>&1 &
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/nginx-ingress" --namespace nginx-ingress-controller > /dev/null 2>&1 &

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

echodate "Verifying nginx-ingress mode deployment..."
retry 10 kubectl wait --for=condition=ready --timeout=1m ingress/kubermatic -n kubermatic
retry 10 check_all_deployments_ready nginx-ingress-controller
echodate "nginx-ingress controller deployed"

echodate "Verifying Gateway API resources do not exist in Ingress mode..."
if kubectl get gateway -n kubermatic kubermatic 2> /dev/null; then
  echodate "ERROR: Gateway should not exist in Ingress mode!"
  exit 1
fi
if kubectl get httproute -n kubermatic kubermatic 2> /dev/null; then
  echodate "ERROR: HTTPRoute should not exist in Ingress mode!"
  exit 1
fi

echodate "Gateway API resources correctly absent"

echodate "Running pre-migration tests (Ingress mode)..."
go_test gateway_api_migration_e2e -timeout 1h -tags e2e -v ./pkg/test/e2e/gateway-api \
  -test.run "TestGatewayAPIPreMigration" \
  -aws-kkp-datacenter "$AWS_E2E_TESTS_DATACENTER" \
  -ssh-pub-key "$(cat "$E2E_SSH_PUBKEY")"

echodate "Pre-migration tests passed"

echodate ""
echodate "Upgrading to Gateway API mode"

REPOSUFFIX=""
if [ "$KUBERMATIC_EDITION" != "ce" ]; then
  REPOSUFFIX="-$KUBERMATIC_EDITION"
fi
KUBERMATIC_VERSION="${KUBERMATIC_VERSION:-$(git rev-parse HEAD)}"

UPGRADE_HELM_VALUES="$(mktemp)"
cat << EOF > $UPGRADE_HELM_VALUES
migrateGatewayAPI: true
kubermaticOperator:
  image:
    repository: "quay.io/kubermatic/kubermatic$REPOSUFFIX"
    tag: "$KUBERMATIC_VERSION"

minio:
  credentials:
    accessKey: test
    secretKey: testtest

telemetry:
  uuid: "559a1b90-b5d0-40aa-a74d-bd9e808ec10f"
  schedule: "* * * * *"
  reporterArgs:
    - stdout
    - --client-uuid=\$(CLIENT_UUID)
    - --record-dir=\$(RECORD_DIR)
EOF

export INSTALLER_FLAGS="--migrate-gateway-api --helm-values $UPGRADE_HELM_VALUES"

echodate "Re-running kubermatic-installer with --migrate-gateway-api flag..."

# Prepare KubermaticConfiguration for upgrade (config from Phase 1 is a temp file, so we recreate)
# Note that this duplicates logic from setup-kubermatic-in-kind.sh because that script's
# KUBERMATIC_CONFIG is a local temp file no longer accessible here.
KUBERMATIC_CONFIG="$(mktemp)"
IMAGE_PULL_SECRET_INLINE="$(echo "$IMAGE_PULL_SECRET_DATA" | base64 --decode | jq --compact-output --monochrome-output '.')"
KUBERMATIC_DOMAIN="${KUBERMATIC_DOMAIN:-ci.kubermatic.io}"

cp $KUBERMATIC_YAML $KUBERMATIC_CONFIG
sed -i "s;__SERVICE_ACCOUNT_KEY__;$SERVICE_ACCOUNT_KEY;g" $KUBERMATIC_CONFIG
sed -i "s;__IMAGE_PULL_SECRET__;$IMAGE_PULL_SECRET_INLINE;g" $KUBERMATIC_CONFIG
sed -i "s;__KUBERMATIC_DOMAIN__;$KUBERMATIC_DOMAIN;g" $KUBERMATIC_CONFIG

./_build/kubermatic-installer deploy kubermatic-master \
  --storageclass copy-default \
  --config "$KUBERMATIC_CONFIG" \
  --helm-values "$UPGRADE_HELM_VALUES" \
  $INSTALLER_FLAGS

echodate "Waiting for Kubermatic Operator to restart with Gateway API enabled..."
sleep 5
retry 10 check_all_deployments_ready kubermatic
echodate "Operator restarted with Gateway API mode"

protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/envoy-gateway" --namespace envoy-gateway-controller > /dev/null 2>&1 &

echodate "Verifying Gateway API resources deployed..."
retry 10 check_all_deployments_ready envoy-gateway-controller
retry 10 check_all_deployments_ready kubermatic
retry 10 kubectl get gatewayclass kubermatic-envoy
retry 10 kubectl get gateway -n kubermatic kubermatic
retry 10 kubectl get httproute -n kubermatic kubermatic
echodate "Gateway API resources deployed"

echodate "Verifying old Ingress was removed..."
if kubectl get ingress -n kubermatic kubermatic 2> /dev/null; then
  echodate "ERROR: Old Ingress still exists after migration!"
  kubectl get ingress -n kubermatic kubermatic -o yaml
  exit 1
fi
echodate "Old Ingress correctly removed"

echodate ""
echodate "Verifying cluster health after migration ==="
echodate "Running post-migration tests (Gateway API mode)..."

go_test gateway_api_migration_e2e -timeout 1h -tags e2e -v ./pkg/test/e2e/gateway-api \
  -test.run "TestGatewayAPIPostMigration" \
  -aws-kkp-datacenter "$AWS_E2E_TESTS_DATACENTER" \
  -ssh-pub-key "$(cat "$E2E_SSH_PUBKEY")"

echodate "Post-migration tests passed"
echodate "Gateway API migration tests completed successfully!"
