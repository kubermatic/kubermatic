#!/usr/bin/env bash

# Copyright 2024 The Kubermatic Kubernetes Platform contributors.
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

cd $(dirname $0)/../..
source hack/lib.sh

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

retry 5 vault_ci_login

backupCredentials="$(vault kv get --format=json dev/e2e-kkp-cluster-backup)"
export BACKUP_ACCESS_KEY_ID="$(echo "$backupCredentials" | jq -r '.data.data.accessKeyID')"
export BACKUP_SECRET_ACCESS_KEY="$(echo "$backupCredentials" | jq -r '.data.data.secretAccessKey')"
export BACKUP_BUCKET="$(echo "$backupCredentials" | jq -r '.data.data.bucket')"
export BACKUP_REGION="$(echo "$backupCredentials" | jq -r '.data.data.region')"

if [ -z "${E2E_SSH_PUBKEY:-}" ]; then
  echodate "Getting default SSH pubkey for machines from Vault"
  E2E_SSH_PUBKEY="$(mktemp)"
  vault kv get -field=pubkey dev/e2e-machine-controller-ssh-key > "${E2E_SSH_PUBKEY}"
else
  E2E_SSH_PUBKEY_CONTENT="${E2E_SSH_PUBKEY}"
  E2E_SSH_PUBKEY="$(mktemp)"
  echo "${E2E_SSH_PUBKEY_CONTENT}" > "${E2E_SSH_PUBKEY}"
fi

echodate "SSH public key will be $(head -c 25 ${E2E_SSH_PUBKEY})...$(tail -c 25 ${E2E_SSH_PUBKEY})"

export KIND_CLUSTER_NAME="${SEED_NAME:-kubermatic}"

source hack/ci/setup-kind-cluster.sh

# gather the logs of all things in the cluster control plane and in the Kubermatic namespace
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/cluster-control-plane" --namespace 'cluster-*' > /dev/null 2>&1 &
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &

source hack/ci/setup-kubermatic-cluster-backup-in-kind.sh

echodate "Running cluster-backup tests..."

go_test cluster_backup_e2e -timeout 120m -tags e2e -v ./pkg/test/e2e/cluster-backup \
  -kubeconfig "$KUBECONFIG" \
  -aws-kkp-datacenter "$AWS_E2E_TESTS_DATACENTER" \
  -ssh-pub-key "$(cat "$E2E_SSH_PUBKEY")"

echodate "Tests completed successfully!"
