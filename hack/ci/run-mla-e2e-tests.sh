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
### couple of test Presets and Users and then runs the MLA e2e tests.

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
source hack/ci/setup-kubermatic-mla-in-kind.sh

echodate "Creating Hetzner preset..."
cat << EOF > preset-hetzner.yaml
apiVersion: kubermatic.k8c.io/v1
kind: Preset
metadata:
  name: e2e-hetzner
  namespace: kubermatic
spec:
  hetzner:
    token: ${HZ_E2E_TOKEN}
EOF
retry 2 kubectl apply -f preset-hetzner.yaml

echodate "Creating roxy-admin user..."
cat << EOF > user.yaml
apiVersion: kubermatic.k8c.io/v1
kind: User
metadata:
  name: roxy-admin
spec:
  admin: true
  email: roxy-admin@kubermatic.com
  name: roxy-admin
EOF
retry 2 kubectl apply -f user.yaml

echodate "Running mla tests..."

go_test mla_e2e -timeout 30m -tags mla -v ./pkg/test/e2e/mla \
  -kubeconfig "$KUBECONFIG" \
  -seed kubermatic \
  -datacenter hetzner-hel1 \
  -preset e2e-hetzner

echodate "Tests completed successfully!"
