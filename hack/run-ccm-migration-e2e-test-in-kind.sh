#!/usr/bin/env bash

# Copyright 2021 The Kubermatic Kubernetes Platform contributors.
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

### This script sets up a local KKP installation in kind, deploys a
### couple of test Presets and Users and then runs the e2e tests for the
### external ccm-migration.

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

export GIT_HEAD_HASH="$(git rev-parse HEAD)"
export KUBERMATIC_VERSION="${GIT_HEAD_HASH}"

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

echodate "Creating kind cluster"
export KIND_CLUSTER_NAME="${SEED_NAME:-kubermatic}"
source hack/ci/setup-kind-cluster.sh

if [ -x "$(command -v protokol)" ]; then
  # gather the logs of all things in the cluster control plane and in the Kubermatic namespace
  protokol --kubeconfig "${HOME}/.kube/config" --flat --output "$ARTIFACTS/logs/cluster-control-plane" --namespace 'cluster-*' > /dev/null 2>&1 &
  protokol --kubeconfig "${HOME}/.kube/config" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &
fi

echodate "Setting up Kubermatic in kind on revision ${KUBERMATIC_VERSION}"

beforeKubermaticSetup=$(nowms)
source hack/ci/setup-kubermatic-in-kind.sh
pushElapsed kind_kubermatic_setup_duration_milliseconds $beforeKubermaticSetup

PROVIDER_TO_TEST="${PROVIDER}"
TIMEOUT=30m

if [[ "$PROVIDER_TO_TEST" == "openstack" ]]; then
  EXTRA_ARGS="-openstack-domain=${OS_DOMAIN}
    -openstack-tenant=${OS_TENANT_NAME}
    -openstack-username=${OS_USERNAME}
    -openstack-password=${OS_PASSWORD}
    -openstack-floating-ip-pool=${OS_FLOATING_IP_POOL}
    -openstack-network=${OS_NETWORK_NAME}
    -openstack-seed-datacenter=syseleven-dbl1
    "
fi

if [[ "$PROVIDER_TO_TEST" == "vsphere" ]]; then
  EXTRA_ARGS="-vsphere-seed-datacenter=vsphere-ger
    -vsphere-username=${VSPHERE_E2E_USERNAME}
    -vsphere-password=${VSPHERE_E2E_PASSWORD}
    "
fi

if [[ "$PROVIDER_TO_TEST" == "azure" ]]; then
  TIMEOUT=45m
  EXTRA_ARGS="-azure-tenant-id=${AZURE_E2E_TESTS_TENANT_ID}
    -azure-subscription-id=${AZURE_E2E_TESTS_SUBSCRIPTION_ID}
    -azure-client-id=${AZURE_E2E_TESTS_CLIENT_ID}
    -azure-client-secret=${AZURE_E2E_TESTS_CLIENT_SECRET}
    -azure-seed-datacenter=azure-westeurope
    "
fi

if [[ "$PROVIDER_TO_TEST" == "aws" ]]; then
  EXTRA_ARGS="-aws-access-key-id=$AWS_E2E_TESTS_KEY_ID
    -aws-secret-access-key=$AWS_E2E_TESTS_SECRET
    -aws-seed-datacenter=aws-eu-central-1a
    "
fi

# run tests
echodate "Running CCM tests..."

# for unknown reasons, log output is not shown live when using
# "/..." in the package expression (the position of the -v flag
# doesn't make a difference).
go_test ccm_migration_${PROVIDER_TO_TEST} \
  -tags=e2e ./pkg/test/e2e/ccm-migration $EXTRA_ARGS \
  -v \
  -timeout 30m \
  -kubeconfig "${HOME}/.kube/config" \
  -provider "$PROVIDER_TO_TEST"
