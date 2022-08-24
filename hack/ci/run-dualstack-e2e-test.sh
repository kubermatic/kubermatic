#!/usr/bin/env bash

# Copyright 2022 The Kubermatic Kubernetes Platform contributors.
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

### This script is used as a postsubmit job and updates the dev master
### cluster after every commit to master.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

function cleanup() {
  if [[ -n "${TMP:-}" ]]; then
    rm -rf "${TMP}"
  fi
}
trap cleanup EXIT SIGINT SIGTERM

export KIND_CLUSTER_NAME="${SEED_NAME:-kubermatic}"
export CNI="${CNI:-}"
export PROVIDER="${PROVIDER:-}"
export KUBERMATIC_YAML=hack/ci/testdata/kubermatic_dualstack.yaml
export WITH_WORKERS=1
source hack/ci/setup-kind-cluster.sh

# gather the logs of all things in the Kubermatic namespace
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &

source hack/ci/setup-kubermatic-in-kind.sh

export GIT_HEAD_HASH="$(git rev-parse HEAD | tr -d '\n')"

echodate "Getting secrets from Vault"
retry 5 vault_ci_login

export AWS_ACCESS_KEY_ID=$(vault kv get -field=accessKeyID dev/e2e-aws)
export AWS_SECRET_ACCESS_KEY=$(vault kv get -field=secretAccessKey dev/e2e-aws)

export AZURE_TENANT_ID="${AZURE_TENANT_ID:-$(vault kv get -field=tenantID dev/e2e-azure)}"
export AZURE_SUBSCRIPTION_ID="${AZURE_SUBSCRIPTION_ID:-$(vault kv get -field=subscriptionID dev/e2e-azure)}"
export AZURE_CLIENT_ID="${AZURE_CLIENT_ID:-$(vault kv get -field=clientID dev/e2e-azure)}"
export AZURE_CLIENT_SECRET="${AZURE_CLIENT_SECRET:-$(vault kv get -field=clientSecret dev/e2e-azure)}"

export GOOGLE_SERVICE_ACCOUNT="$(safebase64 "${GOOGLE_SERVICE_ACCOUNT:-$(vault kv get -field=serviceAccount dev/e2e-gce)}")"

export OS_USERNAME="${OS_USERNAME:-$(vault kv get -field=username dev/syseleven-openstack)}"
export OS_PASSWORD="${OS_PASSWORD:-$(vault kv get -field=password dev/syseleven-openstack)}"
export OS_USER_DOMAIN_NAME="${OS_USER_DOMAIN_NAME:-$(vault kv get -field=OS_USER_DOMAIN_NAME dev/syseleven-openstack)}"
export OS_PROJECT_NAME="${OS_PROJECT_NAME:-$(vault kv get -field=OS_TENANT_NAME dev/syseleven-openstack)}"
export OS_FLOATING_IP_POOL="${OS_FLOATING_IP_POOL:-$(vault kv get -field=OS_FLOATING_IP_POOL dev/syseleven-openstack)}"

export OS_RHEL_USERNAME="${OS_RHEL_USERNAME:-$(vault kv get -field=username dev/redhat-subscription)}"
export OS_RHEL_PASSWORD="${OS_RHEL_PASSWORD:-$(vault kv get -field=password dev/redhat-subscription)}"
export OS_RHEL_OFFLINE_TOKEN="${OS_RHEL_OFFLINE_TOKEN:-$(vault kv get -field=offlineToken dev/redhat-subscription)}"

export HETZNER_TOKEN="${HETZNER_TOKEN:-$(vault kv get -field=token dev/e2e-hetzner)}"

export DO_TOKEN="${DO_TOKEN:-$(vault kv get -field=token dev/e2e-digitalocean)}"

export METAL_AUTH_TOKEN="${METAL_AUTH_TOKEN:-$(vault kv get -field=METAL_AUTH_TOKEN dev/e2e-equinix-metal)}"
export METAL_PROJECT_ID="${METAL_PROJECT_ID:-$(vault kv get -field=METAL_PROJECT_ID dev/e2e-equinix-metal)}"

export VSPHERE_USERNAME="${VSPHERE_USERNAME:-$(vault kv get -field=username dev/e2e-vsphere)}"
export VSPHERE_PASSWORD="${VSPHERE_PASSWORD:-$(vault kv get -field=password dev/e2e-vsphere)}"

echodate "Successfully got secrets for dev from Vault"
echodate "Running dualstack tests..."

export OSNAMES="${OSNAMES:-all}"

go_test dualstack_e2e -race -timeout 90m -tags dualstack -v ./pkg/test/dualstack/... -args --cni $CNI --provider $PROVIDER --os $OSNAMES

echodate "Dualstack tests done."
