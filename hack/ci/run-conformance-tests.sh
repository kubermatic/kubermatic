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
  retry 5 vault_ci_login
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
maxDuration=60 # in minutes
if [[ $provider == "anexia" ]]; then
  EXTRA_ARGS="-anexia-token=${ANEXIA_TOKEN}
    -anexia-template-id=${ANEXIA_TEMPLATE_ID}
    -anexia-vlan-id=${ANEXIA_VLAN_ID}
    -anexia-location-id=${ANEXIA_LOCATION_ID}"
elif [[ $provider == "aws" ]]; then
  EXTRA_ARGS="-aws-access-key-id=${AWS_E2E_TESTS_KEY_ID}
    -aws-secret-access-key=${AWS_E2E_TESTS_SECRET}"
elif [[ $provider == "packet" ]]; then
  maxDuration=90
  EXTRA_ARGS="-packet-api-key=${PACKET_API_KEY}
    -packet-project-id=${PACKET_PROJECT_ID}"
elif [[ $provider == "gcp" ]]; then
  EXTRA_ARGS="-gcp-service-account=${GOOGLE_SERVICE_ACCOUNT}"
elif [[ $provider == "azure" ]]; then
  maxDuration=90
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
    -openstack-project=${OS_TENANT_NAME}
    -openstack-username=${OS_USERNAME}
    -openstack-password=${OS_PASSWORD}"
elif [[ $provider == "vsphere" ]]; then
  EXTRA_ARGS="-vsphere-username=${VSPHERE_E2E_USERNAME}
    -vsphere-password=${VSPHERE_E2E_PASSWORD}
    -vsphere-datastore=HS-FreeNAS"
elif [[ $provider == "kubevirt" ]]; then
  tmpFile="$(mktemp)"
  echo "$KUBEVIRT_E2E_TESTS_KUBECONFIG" > "$tmpFile"
  EXTRA_ARGS="-kubevirt-kubeconfig=$tmpFile"
elif [[ $provider == "alibaba" ]]; then
  EXTRA_ARGS="-alibaba-access-key-id=${ALIBABA_E2E_TESTS_KEY_ID}
    -alibaba-secret-access-key=${ALIBABA_E2E_TESTS_SECRET}"
elif [[ $provider == "nutanix" ]]; then
  EXTRA_ARGS="-nutanix-username=${NUTANIX_E2E_USERNAME}
    -nutanix-password=${NUTANIX_E2E_PASSWORD}
    -nutanix-csi-username=${NUTANIX_E2E_PE_USERNAME}
    -nutanix-csi-password=${NUTANIX_E2E_PE_PASSWORD}
    -nutanix-csi-endpoint=${NUTANIX_E2E_PE_ENDPOINT}
    -nutanix-proxy-url=http://${NUTANIX_E2E_PROXY_USERNAME}:${NUTANIX_E2E_PROXY_PASSWORD}@10.240.20.100:${NUTANIX_E2E_PROXY_PORT}/
    -nutanix-cluster-name=${NUTANIX_E2E_CLUSTER_NAME}
    -nutanix-project-name=${NUTANIX_E2E_PROJECT_NAME}
    -nutanix-subnet-name=${NUTANIX_E2E_SUBNET_NAME}"
elif [[ $provider == "vmware-cloud-director" ]]; then
  EXTRA_ARGS="-vmware-cloud-director-username=${VCD_USER}
    -vmware-cloud-director-password=${VCD_PASSWORD}
    -vmware-cloud-director-organization=${VCD_ORG}
    -vmware-cloud-director-vdc=${VCD_VDC}
    -vmware-cloud-director-ovdc-network=${VCD_OVDC_NETWORK}"
fi

# in periodic jobs, we run multiple scenarios (e.g. testing azure in 1.21 and 1.22),
# so we must multiply the maxDuration with the number of scenarios
numDists=$(echo "${DISTRIBUTIONS:-}" | tr "," "\n" | wc -l)
numVersions=$(echo "${VERSIONS_TO_TEST:-}" | tr "," "\n" | wc -l)
((maxDuration = $numDists * $numVersions * $maxDuration))

# add a bit of setup time to bring up the project, tear it down again etc.
((maxDuration = $maxDuration + 30))

# copy conformance junit into artifacts to process it in Prow
function copy_junit {
    echodate "Copying conformance results to ${ARTIFACTS}"
    cp ${ARTIFACTS}/conformance/* ${ARTIFACTS}/
}
appendTrap copy_junit EXIT

timeout -s 9 "${maxDuration}m" ./_build/conformance-tester $EXTRA_ARGS \
  -client="${SETUP_MODE:-api}" \
  -name-prefix=prow-e2e \
  -kubeconfig=$KUBECONFIG \
  -kubermatic-seed-cluster="$SEED_NAME" \
  -kubermatic-endpoint="$KUBERMATIC_API_ENDPOINT" \
  -kubermatic-nodes=3 \
  -kubermatic-parallel-clusters=1 \
  -reports-root="$ARTIFACTS/conformance" \
  -log-directory="$ARTIFACTS/logs" \
  -create-oidc-token=true \
  -versions="$VERSIONS_TO_TEST" \
  -providers=$provider \
  -node-ssh-pub-key="$E2E_SSH_PUBKEY" \
  -distributions="${DISTRIBUTIONS:-}" \
  -exclude-distributions="${EXCLUDE_DISTRIBUTIONS:-}" \
  -dex-helm-values-file="$KUBERMATIC_DEX_VALUES_FILE" \
  -only-test-creation=${ONLY_TEST_CREATION:-false} \
  -enable-psp=${KUBERMATIC_PSP_ENABLED:-false} \
  -pushgateway-endpoint="pushgateway.monitoring.svc.cluster.local.:9091"
