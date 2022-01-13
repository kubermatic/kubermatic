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

export KUBERMATIC_DEX_VALUES_FILE=${KUBERMATIC_DEX_VALUES_FILE:-$(realpath hack/ci/testdata/konnectivity_oauth_values.yaml)}
export KUBERMATIC_API_ENDPOINT=${KUBERMATIC_API_ENDPOINT:-https://dev.kubermatic.io}

export GIT_HEAD_HASH="$(git rev-parse HEAD | tr -d '\n')"

echodate "Getting secrets from Vault"
retry 5 vault_ci_login

if [ -z "${KUBECONFIG:-}" ]; then
  echodate "No \$KUBECONFIG set, defaulting to Vault key dev/seed-clusters/dev.kubermatic.io"

  KUBECONFIG="$(mktemp)"
  vault kv get -field=kubeconfig dev/seed-clusters/dev.kubermatic.io > $KUBECONFIG
fi

export KUBERMATIC_OIDC_LOGIN=$(vault kv get -field=username dev/e2e-dex)
export KUBERMATIC_OIDC_PASSWORD=$(vault kv get -field=password dev/e2e-dex)
export AWS_ACCESS_KEY_ID=$(vault kv get -field=accessKeyID dev/e2e-aws)
export AWS_SECRET_ACCESS_KEY=$(vault kv get -field=secretAccessKey dev/e2e-aws)

echodate "Successfully got secrets for dev from Vault"

echodate "Running konnectivity tests..."

go test -tags e2e -v ./pkg/test/e2e/konnectivity/...

echodate "Konnecitvity tests done."
