#!/usr/bin/env bash

# Copyright 2025 The Kubermatic Kubernetes Platform contributors.
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

# Directly use environment variables set earlier in the script or passed in.
APPLICATION_INSTALLATION_NAME="${APPLICATION_INSTALLATION_NAME:-}"
APPLICATION_NAME="${APPLICATION_NAME:-}"
APPLICATION_NAMESPACE="${APPLICATION_NAMESPACE:-}"
APP_LABEL_KEY="${APP_LABEL_KEY:-}"
NAMES="${NAMES:-}"

# Ensure all environment variables are set
if [[ -z "$APPLICATION_INSTALLATION_NAME" || -z "$APPLICATION_NAME" || -z "$APPLICATION_NAMESPACE" || -z "$APP_LABEL_KEY" || -z "$NAMES" ]]; then
  echo "One or more required environment variables are missing."
  exit 1
fi

echo "Running test with application name '$APPLICATION_NAME' and version '$APPLICATION_NAME'"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

export KIND_CLUSTER_NAME="${SEED_NAME:-kubermatic}"
export KUBERMATIC_YAML=hack/ci/testdata/kubermatic.yaml
source hack/ci/setup-kind-cluster.sh

# gather the logs of all things in the cluster control plane and in the Kubermatic namespace
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/cluster-control-plane" --namespace 'cluster-*' > /dev/null 2>&1 &
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &

kubectl apply -f pkg/crd/k8c.io
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

echodate "Running $APPLICATION_NAME tests..."

pathToFile="pkg/ee/default-application-catalog/applicationdefinitions/$APPLICATION_NAME-app.yaml"
echodate "File path: $pathToFile"

if [[ -s "$pathToFile" ]]; then
  versions=$(yq eval '.spec.versions[].version' "$pathToFile")

  # Extract the defaultValuesBlock (this will capture everything after 'defaultValuesBlock:' until 'documentationURL')
  defaultValuesBlock=$(yq eval '.spec.defaultValuesBlock' "$pathToFile")

  echodate "Processing default values block: $defaultValuesBlock"

  for version in $versions; do
    echodate "Processing version: $version"
    go_test default_application_catalog_test -timeout 1h -tags e2e -v ./pkg/test/e2e/defaultappcatalog \
      -aws-kkp-datacenter "$AWS_E2E_TESTS_DATACENTER" \
      -ssh-pub-key "$(cat "$E2E_SSH_PUBKEY")" \
      -application-installation-name "$APPLICATION_INSTALLATION_NAME" \
      -application-name "$APPLICATION_NAME" \
      -application-namespace "$APPLICATION_NAMESPACE" \
      -application-version "$version" \
      -app-label-key "$APP_LABEL_KEY" \
      -names "$NAMES" \
      -default-values-block "$defaultValuesBlock"
  done
else
  echo "File is empty or does not exist."
fi

echodate "Application $APPLICATION_NAME tests done."
