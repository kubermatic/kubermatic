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

### Installs the latest patch release of the previous KKP minor with both MLA
### stacks into kind, records MLA data and configuration on an AWS user
### cluster, upgrades everything to this PR and verifies that the data and the
### configuration survived. The old release runs with Helm 3 and the PR with
### Helm 4, the same path customers take.

set -euo pipefail

cd "$(dirname "$0")/../.."
source hack/lib.sh
source hack/ci/lib-gateway-api-migration.sh
source hack/ci/lib-mla-upgrade.sh

: "${KKP_PREVIOUS_HELM_BINARY:=helm3}"
STATE_FILE="${ARTIFACTS:-/tmp}/mla-upgrade-state.json"

TEST_NAME="Pre-warm Go build cache"
beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

if [ -z "${E2E_SSH_PUBKEY:-}" ]; then
  retry 5 vault_ci_login
  E2E_SSH_PUBKEY="$(mktemp)"
  vault kv get -field=pubkey dev/e2e-machine-controller-ssh-key > "${E2E_SSH_PUBKEY}"
fi

export SEED_NAME="${SEED_NAME:-kubermatic}"
export BUILD_ID="${BUILD_ID:-$(date +%s)}"
export KIND_CLUSTER_NAME="$SEED_NAME"
source hack/ci/setup-kind-cluster.sh

for ns in kubermatic mla monitoring logging; do
  protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/$ns" --namespace "$ns" > /dev/null 2>&1 &
done
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/cluster-control-plane" --namespace 'cluster-*' > /dev/null 2>&1 &

TEST_NAME="Resolve previous release"
PREVIOUS_VERSION="$(resolve_previous_minor_release)"
PREVIOUS_DIR="/tmp/kkp-${PREVIOUS_VERSION}"
echodate "Upgrading from ${PREVIOUS_VERSION}."
download_kkp_release "${PREVIOUS_VERSION}" "${PREVIOUS_DIR}"

KUBERMATIC_CONFIG="$(mktemp)"
export KUBERMATIC_CONFIG
render_kubermatic_config "$KUBERMATIC_CONFIG"

PREVIOUS_VALUES="$(mktemp)"
write_operator_values "$PREVIOUS_VALUES" "${PREVIOUS_VERSION}"
deploy_mla_stacks "previous release" "${PREVIOUS_DIR}/kubermatic-installer" "${PREVIOUS_DIR}/charts" "$PREVIOUS_VALUES" --helm-binary "$KKP_PREVIOUS_HELM_BINARY"

# delete user clusters even when a step below fails, KKP's own cleanup may not
# work after a broken upgrade
appendTrap cleanup_kubermatic_clusters_in_kind EXIT

TEST_NAME="Pre-upgrade MLA checks"
ginkgo_test mla_upgrade_pre --tags mlaupgrade --label-filter pre-upgrade --timeout 60m --no-color -v ./pkg/test/e2e/mlaupgrade -- \
  -kubeconfig "$KUBECONFIG" \
  -aws-kkp-datacenter "$AWS_E2E_TESTS_DATACENTER" \
  -ssh-pub-key "$(cat "$E2E_SSH_PUBKEY")" \
  -cluster-version "${MLA_UPGRADE_CLUSTER_VERSION:-}" \
  -upgrade-state-file "$STATE_FILE"

if [ "${MLA_UPGRADE_CHAOS_DELETE_MINIO:-}" = "1" ]; then
  echodate "Negative test: deleting MinIO and its volume before the upgrade."
  kubectl --namespace mla delete deployment minio --wait=true
  kubectl --namespace mla delete pvc --selector app=minio --wait=true
fi

export KUBERMATIC_VERSION="${KUBERMATIC_VERSION:-$(git rev-parse HEAD)}"
build_kkp_under_test

PR_VALUES="$(mktemp)"
write_operator_values "$PR_VALUES" "${KUBERMATIC_VERSION}"
deploy_mla_stacks "this branch" ./_build/kubermatic-installer charts "$PR_VALUES"

TEST_NAME="Post-upgrade MLA checks"
ginkgo_test mla_upgrade_post --tags mlaupgrade --label-filter post-upgrade --timeout 50m --no-color -v ./pkg/test/e2e/mlaupgrade -- \
  -kubeconfig "$KUBECONFIG" \
  -upgrade-state-file "$STATE_FILE"

echodate "MLA upgrade e2e passed."
