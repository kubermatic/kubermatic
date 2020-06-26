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

set -euo pipefail

## CI run conformance tester
source hack/lib.sh

### Defaults
export VERSIONS=${VERSIONS_TO_TEST:-"v1.12.4"}
export EXCLUDE_DISTRIBUTIONS=${EXCLUDE_DISTRIBUTIONS:-ubuntu,centos,sles,rhel}
export ONLY_TEST_CREATION=${ONLY_TEST_CREATION:-false}
provider=${PROVIDER:-"aws"}
export WORKER_NAME=${BUILD_ID}
if [[ "${KUBERMATIC_NO_WORKER_NAME:-}" = "true" ]]; then
  WORKER_NAME=""
fi

mkdir -p $HOME/.ssh
if ! [[ -e $HOME/.ssh/id_rsa.pub ]]; then
  echo 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCo3amVmCkIZo4cgj2kjU2arZKlzzOhaOuveH9aJbL4mlVHVsEcVk+RSty4AMK1GQL3+Ii7iGicKWwge4yefc75aOtUncfF01rnBsNvi3lOqJR/6POHy4OnPXJElvEn7jii/pAUeyr8halBezQTUkvRiUtlJo6oEb2dRN5ujyFm5TuIxgM0UFVGBRoD0agGr87GaQsUahf+PE1zHEid+qQPz7EdMo8/eRNtgikhBG1/ae6xRstAi0QU8EgjKvK1ROXOYTlpTBFElApOXZacH91WvG0xgPnyxIXoKtiCCNGeu/0EqDAgiXfmD2HK/WAXwJNwcmRvBaedQUS4H0lNmvj5' \
    > $HOME/.ssh/id_rsa.pub
fi
chmod 0700 $HOME/.ssh

if [[ -n ${OPENSHIFT:-} ]]; then
  OPENSHIFT_ARG="-openshift=true"
  export VERSIONS=${OPENSHIFT_VERSION}
fi

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
  EXTRA_ARGS="-kubevirt-kubeconfig=${KUBEVIRT_E2E_TESTS_KUBECONFIG}"
elif [[ $provider == "alibaba" ]]; then
  EXTRA_ARGS="-alibaba-access-key-id=${ALIBABA_E2E_TESTS_KEY_ID}
     -alibaba-secret-access-key=${ALIBABA_E2E_TESTS_SECRET}"
fi

# Needed when running in kind
which kind && EXTRA_ARGS="$EXTRA_ARGS -create-oidc-token=true"

kubermatic_delete_cluster="true"
if [ -n "${UPGRADE_TEST_BASE_HASH:-}" ]; then
  kubermatic_delete_cluster="false"
fi

timeout -s 9 90m ./_build/conformance-tests ${EXTRA_ARGS:-} \
  -debug \
  -worker-name=${WORKER_NAME} \
  -kubeconfig=$KUBECONFIG \
  -kubermatic-nodes=3 \
  -kubermatic-parallel-clusters=1 \
  -name-prefix=prow-e2e \
  -reports-root=/reports \
  -cleanup-on-start=false \
  -run-kubermatic-controller-manager=false \
  -versions="$VERSIONS" \
  -providers=$provider \
  -only-test-creation="${ONLY_TEST_CREATION}" \
  -exclude-distributions="${EXCLUDE_DISTRIBUTIONS}" \
  ${OPENSHIFT_ARG:-} \
  -kubermatic-delete-cluster=${kubermatic_delete_cluster} \
  -print-ginkgo-logs=true \
  -pushgateway-endpoint="pushgateway.monitoring.svc.cluster.local.:9091"
