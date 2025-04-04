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

download_kube_test() {
  local directory="$1"

  echodate "Kubernetes release: $RELEASES_TO_TEST"

  KUBE_VERSION="$(download_archive https://dl.k8s.io/release/stable-$RELEASES_TO_TEST.txt -Ls)"
  echodate "Kubernetes version: $KUBE_VERSION"

  TMP_DIR="/tmp/k8s"

  rm -rf -- "$directory"
  mkdir -p "$TMP_DIR" "$directory"

  directory="$directory/platforms/$(go env GOOS)/$(go env GOARCH)"
  mkdir -p "$directory"

  TEST_BINARIES=kubernetes-test-$(go env GOOS)-$(go env GOARCH).tar.gz
  mkdir -p "$TMP_DIR"
  download_archive "https://dl.k8s.io/$KUBE_VERSION/$TEST_BINARIES" -Lo "$TMP_DIR/$TEST_BINARIES"
  tar -zxf "$TMP_DIR/$TEST_BINARIES" -C "$TMP_DIR"
  mv $TMP_DIR/kubernetes/test/bin/* "$directory/"
  rm -rf -- "$TMP_DIR"

  CLIENT_BINARIES=kubernetes-client-$(go env GOOS)-$(go env GOARCH).tar.gz
  mkdir -p "$TMP_DIR"
  download_archive "https://dl.k8s.io/$KUBE_VERSION/$CLIENT_BINARIES" -Lo "$TMP_DIR/$CLIENT_BINARIES"
  tar -zxf "$TMP_DIR/$CLIENT_BINARIES" -C "$TMP_DIR"
  mv $TMP_DIR/kubernetes/client/bin/* "$directory/"
  rm -rf -- "$TMP_DIR"

  echodate "Done downloading Kubernetes test binaries."
}

TEST_NAME="Download kube-test binaries"
echodate "Downloading kube-test binaries..."

download_kube_test "/opt/kube-test/$RELEASES_TO_TEST"

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

if provider_disabled $provider; then
  exit 0
fi

if [[ $provider == "anexia" ]]; then
  EXTRA_ARGS="-anexia-token=${ANEXIA_TOKEN}
    -anexia-template-id=${ANEXIA_TEMPLATE_ID}
    -anexia-vlan-id=${ANEXIA_VLAN_ID}
    -anexia-kkp-datacenter=anexia-at"
elif [[ $provider == "aws" ]]; then
  EXTRA_ARGS="-aws-access-key-id=${AWS_E2E_TESTS_KEY_ID}
    -aws-secret-access-key=${AWS_E2E_TESTS_SECRET}
    -aws-kkp-datacenter=aws-eu-west-1a"
elif [[ $provider == "packet" ]]; then
  maxDuration=90
  EXTRA_ARGS="-packet-api-key=${PACKET_API_KEY}
    -packet-project-id=${PACKET_PROJECT_ID}
    -packet-kkp-datacenter=packet-am"
elif [[ $provider == "gcp" ]]; then
  EXTRA_ARGS="-gcp-service-account=$(safebase64 "$GOOGLE_SERVICE_ACCOUNT")
    -gcp-kkp-datacenter=gcp-westeurope"
elif [[ $provider == "azure" ]]; then
  maxDuration=90
  EXTRA_ARGS="-azure-client-id=${AZURE_E2E_TESTS_CLIENT_ID}
    -azure-client-secret=${AZURE_E2E_TESTS_CLIENT_SECRET}
    -azure-tenant-id=${AZURE_E2E_TESTS_TENANT_ID}
    -azure-subscription-id=${AZURE_E2E_TESTS_SUBSCRIPTION_ID}
    -azure-kkp-datacenter=azure-westeurope"
elif [[ $provider == "digitalocean" ]]; then
  EXTRA_ARGS="-digitalocean-token=${DO_E2E_TESTS_TOKEN}
    -digitalocean-kkp-datacenter=do-ams3"
elif [[ $provider == "hetzner" ]]; then
  EXTRA_ARGS="-hetzner-token=${HZ_E2E_TOKEN}
    -hetzner-kkp-datacenter=hetzner-nbg1"
elif [[ $provider == "openstack" ]]; then
  EXTRA_ARGS="-openstack-domain=${OS_DOMAIN}
    -openstack-project=${OS_TENANT_NAME}
    -openstack-username=${OS_USERNAME}
    -openstack-password=${OS_PASSWORD}
    -openstack-kkp-datacenter=syseleven-dbl1"
elif [[ $provider == "vsphere" ]]; then
  EXTRA_ARGS="-vsphere-username=${VSPHERE_E2E_USERNAME}
    -vsphere-password=${VSPHERE_E2E_PASSWORD}
    -vsphere-kkp-datacenter=vsphere-ger"
elif [[ $provider == "kubevirt" ]]; then
  maxDuration=90
  tmpFile="$(mktemp)"
  echo "$KUBEVIRT_E2E_TESTS_KUBECONFIG" > "$tmpFile"
  EXTRA_ARGS="-kubevirt-kubeconfig=$tmpFile
    -kubevirt-kkp-datacenter=kubevirt-europe-west3-c"
elif [[ $provider == "alibaba" ]]; then
  EXTRA_ARGS="-alibaba-access-key-id=${ALIBABA_E2E_TESTS_KEY_ID}
    -alibaba-secret-access-key=${ALIBABA_E2E_TESTS_SECRET}
    -alibaba-kkp-datacenter=alibaba-eu-central-1a"
elif [[ $provider == "nutanix" ]]; then
  EXTRA_ARGS="-nutanix-username=${NUTANIX_E2E_USERNAME}
    -nutanix-password=${NUTANIX_E2E_PASSWORD}
    -nutanix-csi-username=${NUTANIX_E2E_PE_USERNAME}
    -nutanix-csi-password=${NUTANIX_E2E_PE_PASSWORD}
    -nutanix-csi-endpoint=${NUTANIX_E2E_PE_ENDPOINT}
    -nutanix-cluster-name=${NUTANIX_E2E_CLUSTER_NAME}
    -nutanix-project-name=${NUTANIX_E2E_PROJECT_NAME}
    -nutanix-subnet-name=${NUTANIX_E2E_SUBNET_NAME}
    -nutanix-kkp-datacenter=nutanix-ger"
elif [[ $provider == "vmwareclouddirector" ]]; then
  EXTRA_ARGS="-vmware-cloud-director-username=${VCD_USER}
    -vmware-cloud-director-password=${VCD_PASSWORD}
    -vmware-cloud-director-organization=${VCD_ORG}
    -vmware-cloud-director-vdc=${VCD_VDC}
    -vmware-cloud-director-ovdc-networks=${VCD_OVDC_NETWORK}
    -vmware-cloud-director-kkp-datacenter=vmware-cloud-director-ger"
fi

# in periodic jobs, we run multiple scenarios (e.g. testing azure in 1.21 and 1.22),
# so we must multiply the maxDuration with the number of scenarios
numDists=$(echo "${DISTRIBUTIONS:-}" | tr "," "\n" | wc -l)
numReleases=$(echo "${RELEASES_TO_TEST:-}" | tr "," "\n" | wc -l)
((maxDuration = $numDists * $numReleases * $maxDuration))

# add a bit of setup time to bring up the project, tear it down again etc.
((maxDuration = $maxDuration + 30))

# copy conformance junit into artifacts to process it in Prow
function copy_junit {
  echodate "Copying conformance results to ${ARTIFACTS}"
  cp -r ${ARTIFACTS}/conformance/junit.*.xml ${ARTIFACTS}/
}
appendTrap copy_junit EXIT

timeout -s 9 "${maxDuration}m" ./_build/conformance-tester $EXTRA_ARGS \
  -name-prefix="kkp-$BUILD_ID" \
  -kubeconfig=$KUBECONFIG \
  -kubermatic-seed-cluster="$SEED_NAME" \
  -kubermatic-nodes=3 \
  -kubermatic-parallel-clusters=1 \
  -reports-root="$ARTIFACTS/conformance" \
  -log-directory="$ARTIFACTS/logs" \
  -releases="${RELEASES_TO_TEST:-}" \
  -providers=$provider \
  -node-ssh-pub-key="$E2E_SSH_PUBKEY" \
  -distributions="${DISTRIBUTIONS:-}" \
  -exclude-distributions="${EXCLUDE_DISTRIBUTIONS:-}" \
  -exclude-tests="${EXCLUDE_TESTS:-}" \
  -scenario-options="${SCENARIO_OPTIONS:-}" \
  -pushgateway-endpoint="pushgateway.monitoring.svc.cluster.local.:9091" \
  -results-file "$ARTIFACTS/conformance-tester-results.json"
