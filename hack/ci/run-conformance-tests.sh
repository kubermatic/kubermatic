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

### After having set up a local KKP installation, this script is then
### used to run the conformance-tester for a given cloud provider.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

if [ -z "${E2E_SSH_PUBKEY:-}" ]; then
  echodate "Getting default SSH pubkey for machines from Vault"
  export VAULT_ADDR=https://vault.loodse.com/
  retry 5 vault write \
    --format=json auth/approle/login \
    role_id=${VAULT_ROLE_ID} secret_id=${VAULT_SECRET_ID} > /tmp/vault-token-response.json

  E2E_SSH_PUBKEY="$(mktemp)"
  vault kv get -field=pubkey dev/e2e-machine-controller-ssh-key > "${E2E_SSH_PUBKEY}"
else
  E2E_SSH_PUBKEY_CONTENT="${E2E_SSH_PUBKEY}"
  E2E_SSH_PUBKEY="$(mktemp)"
  echo "${E2E_SSH_PUBKEY_CONTENT}" > "${E2E_SSH_PUBKEY}"
fi

echodate "SSH public key will be $(head -c 25 ${E2E_SSH_PUBKEY})...$(tail -c 25 ${E2E_SSH_PUBKEY})"

EXTRA_ARGS=""
provider="${PROVIDER:-aws}"
if [[ $provider == "aws" ]]; then
  EXTRA_ARGS="-aws-access-key-id=${AWS_E2E_TESTS_KEY_ID}
    -aws-secret-access-key=${AWS_E2E_TESTS_SECRET}"
elif [[ $provider == "packet" ]]; then
  EXTRA_ARGS="-packet-api-key=${PACKET_API_KEY}
    -packet-project-id=${PACKET_PROJECT_ID}"
elif [[ $provider == "gcp" ]]; then
  EXTRA_ARGS="-gcp-service-account=${GOOGLE_SERVICE_ACCOUNT}"
elif [[ $provider == "azure" ]]; then
  EXTRA_ARGS="-azure-client-id=${AZURE_E2E_TESTS_CLIENT_ID}
    -azure-client-secret=${AZURE_E2E_TESTS_CLIENT_SECRET}
    -azure-tenant-id=${AZURE_E2E_TESTS_TENANT_ID}
    -azure-subscription-id=${AZURE_E2E_TESTS_SUBSCRIPTION_ID}"
elif [[ $provider == "digitalocean" ]]; then
  EXTRA_ARGS="-digitalocean-token=${DO_E2E_TESTS_TOKEN}"
elif [[ $provider == "hetzner" ]]; then
  EXTRA_ARGS="-hetzner-token=${HZ_E2E_TOKEN}"
elif [[ $provider == "openstack" ]]; then
  EXTRA_ARGS="-openstack-domain=${OS_DOMAIN}
    -openstack-tenant=${OS_TENANT_NAME}
    -openstack-username=${OS_USERNAME}
    -openstack-password=${OS_PASSWORD}"
elif [[ $provider == "vsphere" ]]; then
  EXTRA_ARGS="-vsphere-username=${VSPHERE_E2E_USERNAME}
    -vsphere-password=${VSPHERE_E2E_PASSWORD}"
elif [[ $provider == "kubevirt" ]]; then
  tmpFile="$(mktemp)"
  echo "$KUBEVIRT_E2E_TESTS_KUBECONFIG" > "$tmpFile"
  EXTRA_ARGS="-kubevirt-kubeconfig=$tmpFile"
elif [[ $provider == "alibaba" ]]; then
  EXTRA_ARGS="-alibaba-access-key-id=${ALIBABA_E2E_TESTS_KEY_ID}
    -alibaba-secret-access-key=${ALIBABA_E2E_TESTS_SECRET}"
fi

timeout -s 9 90m ./_build/conformance-tests $EXTRA_ARGS \
  -name-prefix=prow-e2e \
  -kubeconfig=$KUBECONFIG \
  -kubermatic-seed-cluster="$SEED_NAME" \
  -kubermatic-endpoint="$KUBERMATIC_API_ENDPOINT" \
  -kubermatic-nodes=3 \
  -kubermatic-parallel-clusters=1 \
  -reports-root=/reports \
  -create-oidc-token=true \
  -versions="$VERSIONS_TO_TEST" \
  -providers=$provider \
  -node-ssh-pub-key="$E2E_SSH_PUBKEY" \
  -distributions="${DISTRIBUTIONS:-}" \
  -exclude-distributions="${EXCLUDE_DISTRIBUTIONS:-}" \
  -dex-helm-values-file="$KUBERMATIC_DEX_VALUES_FILE" \
  -only-test-creation=${ONLY_TEST_CREATION:-false} \
  -enable-psp=${KUBERMATIC_PSP_ENABLED:-false} \
  -print-ginkgo-logs=true \
  -print-container-logs=true \
  -pushgateway-endpoint="pushgateway.monitoring.svc.cluster.local.:9091"
