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

### This script sets up a local KKP installation in kind once and runs the
### shared e2e pipeline package against it.

set -euo pipefail

cd "$(dirname "$0")/../.."
source hack/lib.sh

echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds "$beforeGocache"

export KIND_CLUSTER_NAME="${SEED_NAME:-kubermatic}"

echodate "Setting up kind cluster..."
source hack/ci/setup-kind-cluster.sh

# gather the logs of all things in the cluster control plane and in the Kubermatic namespace
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/cluster-control-plane" --namespace 'cluster-*' > /dev/null 2>&1 &
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &

echodate "Setting up KKP in kind cluster..."
source hack/ci/setup-kubermatic-in-kind.sh

echodate "Running pipeline e2e tests..."
# -with-user-cluster provisions the shared BYO base cluster for Tier B/C1 feature tests
# (e.g. the Cilium NodeLocalDNS exclude-local-address test). Seed-only Tier A tests run
# regardless.
go_test pipeline -with-user-cluster -count=1 -timeout 30m -tags e2e -p 1 -v ./pkg/test/e2e/pipeline

echodate "Pipeline e2e tests completed successfully!"
