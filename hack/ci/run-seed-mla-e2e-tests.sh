#!/usr/bin/env bash

# Copyright 2026 The Kubermatic Kubernetes Platform contributors.
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

### Installs KKP, a Seed, the usercluster-mla stack and the full seed-mla
### stack into kind and runs the seed MLA e2e suite. No cloud credentials
### are needed.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

export KIND_CLUSTER_NAME="${SEED_NAME:-kubermatic}"

source hack/ci/setup-kind-cluster.sh

for ns in kubermatic mla monitoring logging; do
  protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/$ns" --namespace "$ns" > /dev/null 2>&1 &
done

source hack/ci/setup-kubermatic-mla-in-kind.sh

TEST_NAME="Deploy seed MLA"
echodate "Deploying the seed-mla stack..."

./_build/kubermatic-installer deploy seed-mla \
  --config "$KUBERMATIC_CONFIG" \
  --helm-values hack/ci/testdata/seed-mla-values.yaml

retry 8 check_all_deployments_ready monitoring
# logging has no Deployments, only the Loki StatefulSet and the Alloy DaemonSet
kubectl --namespace logging rollout status statefulset/loki --timeout 5m
kubectl --namespace logging rollout status daemonset/alloy --timeout 5m

echodate "Seed MLA is ready."

TEST_NAME="Run seed MLA e2e tests"
echodate "Running seed MLA tests..."

# this job creates no user cluster, the MLA upgrade job runs those checks
ginkgo_test seed_mla_e2e --tags seedmla --procs 2 --timeout 30m --no-color -v --label-filter "!needs-user-cluster" ./pkg/test/e2e/seedmla -- \
  -kubeconfig "$KUBECONFIG"

echodate "Tests completed successfully!"
