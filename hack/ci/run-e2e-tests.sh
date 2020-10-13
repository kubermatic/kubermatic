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

### This script sets up a local KKP installation in kind and then
### runs the conformance-tester to create userclusters and check their
### Kubernetes conformance.

set -euo pipefail

cd $(dirname $0)/../..
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
if [ -n "${USE_LEGACY_HELM_CHART:-}" ]; then
  source hack/ci/setup-legacy-kubermatic-in-kind.sh
else
  source hack/ci/setup-kubermatic-in-kind.sh
fi
pushElapsed kind_kubermatic_setup_duration_milliseconds $beforeKubermaticSetup

echodate "Running conformance tests"
./hack/ci/run-conformance-tests.sh
