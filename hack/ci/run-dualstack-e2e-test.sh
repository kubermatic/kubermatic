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
export KUBERMATIC_YAML=hack/ci/testdata/kubermatic_dualstack.yaml
export KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ee}"
export WITH_WORKERS=1
source hack/ci/setup-kind-cluster.sh

# gather the logs of all things in the cluster control plane and in the Kubermatic namespace
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/cluster-control-plane" --namespace 'cluster-*' > /dev/null 2>&1 &
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &

source hack/ci/setup-kubermatic-in-kind.sh

echodate "Getting secrets from Vault"
retry 5 vault_ci_login

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

export AWS_E2E_TESTS_KEY_ID=$(vault kv get -field=accessKeyID dev/e2e-aws-kkp)
export AWS_E2E_TESTS_SECRET=$(vault kv get -field=secretAccessKey dev/e2e-aws-kkp)

export ALIBABA_ACCESS_KEY_ID="${ALIBABA_ACCESS_KEY_ID:-$(vault kv get -field=AccessKeyId dev/e2e-alibaba)}"
export ALIBABA_ACCESS_KEY_SECRET="${ALIBABA_ACCESS_KEY_SECRET:-$(vault kv get -field=AccessKeySecret dev/e2e-alibaba)}"

export AZURE_E2E_TESTS_TENANT_ID="${AZURE_E2E_TESTS_TENANT_ID:-$(vault kv get -field=tenantID dev/e2e-azure)}"
export AZURE_E2E_TESTS_SUBSCRIPTION_ID="${AZURE_E2E_TESTS_SUBSCRIPTION_ID:-$(vault kv get -field=subscriptionID dev/e2e-azure)}"
export AZURE_E2E_TESTS_CLIENT_ID="${AZURE_E2E_TESTS_CLIENT_ID:-$(vault kv get -field=clientID dev/e2e-azure)}"
export AZURE_E2E_TESTS_CLIENT_SECRET="${AZURE_E2E_TESTS_CLIENT_SECRET:-$(vault kv get -field=clientSecret dev/e2e-azure)}"

export GOOGLE_SERVICE_ACCOUNT="$(safebase64 "${GOOGLE_SERVICE_ACCOUNT:-$(vault kv get -field=serviceAccount dev/e2e-gce)}")"

export OS_USERNAME="${OS_USERNAME:-$(vault kv get -field=username dev/syseleven-openstack)}"
export OS_PASSWORD="${OS_PASSWORD:-$(vault kv get -field=password dev/syseleven-openstack)}"
export OS_DOMAIN="${OS_DOMAIN:-$(vault kv get -field=OS_USER_DOMAIN_NAME dev/syseleven-openstack)}"
export OS_TENANT_NAME="${OS_TENANT_NAME:-$(vault kv get -field=OS_TENANT_NAME dev/syseleven-openstack)}"
export OS_FLOATING_IP_POOL="${OS_FLOATING_IP_POOL:-$(vault kv get -field=OS_FLOATING_IP_POOL dev/syseleven-openstack)}"

export OS_RHEL_USERNAME="${OS_RHEL_USERNAME:-$(vault kv get -field=user dev/redhat-subscription)}"
export OS_RHEL_PASSWORD="${OS_RHEL_PASSWORD:-$(vault kv get -field=password dev/redhat-subscription)}"
export OS_RHEL_OFFLINE_TOKEN="${OS_RHEL_OFFLINE_TOKEN:-$(vault kv get -field=offlineToken dev/redhat-subscription)}"

export HZ_TOKEN="${HZ_TOKEN:-$(vault kv get -field=token dev/e2e-hetzner)}"

export DO_TOKEN="${DO_TOKEN:-$(vault kv get -field=token dev/e2e-digitalocean)}"

export VSPHERE_E2E_USERNAME="${VSPHERE_E2E_USERNAME:-$(vault kv get -field=username dev/e2e-vsphere)}"
export VSPHERE_E2E_PASSWORD="${VSPHERE_E2E_PASSWORD:-$(vault kv get -field=password dev/e2e-vsphere)}"

if provider_disabled $PROVIDER; then
  exit 0
fi

echodate "Successfully got secrets from Vault"
echodate "Running dualstack tests..."

go_test dualstack_e2e -race -timeout 90m -tags "dualstack,$KUBERMATIC_EDITION" -v ./pkg/test/dualstack \
  -cni "${CNI:-}" \
  -provider "${PROVIDER:-}" \
  -os "${OSNAMES:-all}" \
  -alibaba-kkp-datacenter alibaba-eu-central-1a \
  -aws-kkp-datacenter aws-eu-west-1a \
  -azure-kkp-datacenter azure-westeurope \
  -digitalocean-kkp-datacenter do-fra1 \
  -gcp-kkp-datacenter gcp-westeurope \
  -hetzner-kkp-datacenter hetzner-nbg1 \
  -openstack-kkp-datacenter syseleven-fes1 \
  -vsphere-kkp-datacenter vsphere-ger \
  -ssh-pub-key "$(cat "$E2E_SSH_PUBKEY")"

echodate "Dualstack tests done."
