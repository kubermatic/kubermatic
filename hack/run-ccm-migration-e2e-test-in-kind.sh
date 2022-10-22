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
export KUBERMATIC_YAML="hack/ci/testdata/kubermatic_headless.yaml"

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

case "$PROVIDER_TO_TEST" in
openstack)
  EXTRA_ARGS="-openstack-kkp-datacenter=syseleven-dbl1"
  ;;
vsphere)
  EXTRA_ARGS="-vsphere-kkp-datacenter=vsphere-ger"
  ;;
azure)
  TIMEOUT=45m
  EXTRA_ARGS="-azure-kkp-datacenter=azure-westeurope"
  ;;
aws)
  # default version is 1.23, but AWS CCM requires 1.24, so we must explicitly
  # ask for a 1.24.x cluster
  EXTRA_ARGS="-aws-kkp-datacenter=aws-eu-central-1a -cluster-version=1.24"
  ;;
esac

# run tests
echodate "Running CCM tests..."

# for unknown reasons, log output is not shown live when using
# "/..." in the package expression (the position of the -v flag
# doesn't make a difference).
go_test ccm_migration_${PROVIDER_TO_TEST} \
  -tags=e2e ./pkg/test/e2e/ccm-migration $EXTRA_ARGS \
  -v \
  -timeout $TIMEOUT \
  -kubeconfig "${HOME}/.kube/config" \
  -provider "$PROVIDER_TO_TEST"
