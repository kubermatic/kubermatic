#!/usr/bin/env bash

# Copyright 2022 The Kubermatic Kubernetes Platform contributors.
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

### This script is used as a postsubmit job and updates the dev master
### cluster after every commit to master.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

function cleanup() {
  if [[ -n "${TMP:-}" ]]; then
    rm -rf "${TMP}"
  fi
}
trap cleanup EXIT SIGINT SIGTERM

export KIND_CLUSTER_NAME="${SEED_NAME:-kubermatic}"

source hack/ci/setup-kind-cluster.sh
source hack/ci/setup-kubermatic-in-kind.sh

export GIT_HEAD_HASH="$(git rev-parse HEAD | tr -d '\n')"

echodate "Getting secrets from Vault"
retry 5 vault_ci_login

if [ -z "${KUBECONFIG:-}" ]; then
  echodate "No \$KUBECONFIG set, defaulting to Vault key dev/seed-clusters/dev.kubermatic.io"

  KUBECONFIG="$(mktemp)"
  vault kv get -field=kubeconfig dev/seed-clusters/dev.kubermatic.io > $KUBECONFIG
fi

export AWS_ACCESS_KEY_ID=$(vault kv get -field=accessKeyID dev/e2e-aws)
export AWS_SECRET_ACCESS_KEY=$(vault kv get -field=secretAccessKey dev/e2e-aws)

echodate "Successfully got secrets for dev from Vault"

function print_kubermatic_logs {
  if [[ $? -ne 0 ]]; then
    echodate "Printing logs for Kubermatic API"
    kubectl -n kubermatic logs --tail=-1 --selector='app.kubernetes.io/name=kubermatic-api'
    echodate "Printing logs for Master Controller Manager"
    kubectl -n kubermatic logs --tail=-1 --selector='app.kubernetes.io/name=kubermatic-master-controller-manager'
    echodate "Printing logs for Seed Controller Manager"
    kubectl -n kubermatic logs --tail=-1 --selector='app.kubernetes.io/name=kubermatic-seed-controller-manager'
  fi
}
appendTrap print_kubermatic_logs EXIT

echodate "Running konnectivity tests..."

go test -tags e2e -v ./pkg/test/e2e/konnectivity/... -kubeconfig $KUBECONFIG

echodate "Konnecitvity tests done."
