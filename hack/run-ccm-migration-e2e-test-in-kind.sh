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

echodate "Setting up Kubermatic in kind on revision ${KUBERMATIC_VERSION}"

beforeKubermaticSetup=$(nowms)
source hack/ci/setup-kubermatic-in-kind.sh
pushElapsed kind_kubermatic_setup_duration_milliseconds $beforeKubermaticSetup

export PROVIDER_TO_TEST="${PROVIDER}"
if [[ "$PROVIDER_TO_TEST" == "openstack" ]]; then
  export EXTRA_ARGS="-openstack-domain=${OS_DOMAIN}
    -openstack-tenant=${OS_TENANT_NAME}
    -openstack-username=${OS_USERNAME}
    -openstack-password=${OS_PASSWORD}
    -openstack-auth-url=${OS_AUTH_URL}
    -openstack-region=${OS_REGION}
    -openstack-floating-ip-pool=${OS_FLOATING_IP_POOL}
    -openstack-network=${OS_NETWORK_NAME}
    -openstack-seed-datacenter=syseleven-dbl1
    "
fi

if [[ "$PROVIDER_TO_TEST" == "vsphere" ]]; then
  export EXTRA_ARGS="-vsphere-seed-datacenter=vsphere-ger
    -vsphere-datacenter=dc-1
    -vsphere-cluster=cl-1
    -vsphere-auth-url=${VSPHERE_E2E_ADDRESS}
    -vsphere-username=${VSPHERE_E2E_USERNAME}
    -vsphere-password=${VSPHERE_E2E_PASSWORD}
    "
fi

# run tests
# use ginkgo binary by preference to have better output:
# https://github.com/onsi/ginkgo/issues/633
if [ -x "$(command -v ginkgo)" ]; then
  ginkgo --tags=e2e -v pkg/test/e2e/ccm-migration/ $EXTRA_ARGS \
    -r \
    --randomizeAllSpecs \
    --randomizeSuites \
    --failOnPending \
    --timeout=30m \
    --cover \
    --trace \
    --race \
    --progress \
    -v \
    -- --kubeconfig "${HOME}/.kube/config" \
    --debug-log \
    --provider "${PROVIDER_TO_TEST}"
else
  CGO_ENABLED=1 go test --tags=e2e -v -race ./pkg/test/e2e/ccm-migration/... $EXTRA_ARGS \
    --ginkgo.randomizeAllSpecs \
    --ginkgo.failOnPending \
    --ginkgo.trace \
    --ginkgo.progress \
    --ginkgo.v \
    --timeout=30m \
    --kubeconfig "${HOME}/.kube/config" \
    --debug-log \
    --provider "${PROVIDER_TO_TEST}"
fi
