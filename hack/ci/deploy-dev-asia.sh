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

### This script is used as a postsubmit job and updates the dev-asia seed
### cluster after every commit to master.

set -euo pipefail

cd $(dirname $0)/../..
source ./hack/lib.sh

export DEPLOY_STACK=${DEPLOY_STACK:-kubermatic}
export GIT_HEAD_HASH="$(git rev-parse HEAD | tr -d '\n')"

# This job does not need to build any binaries, as we only install a few
# CRDs and charts to this seed cluster.

echodate "Getting secrets from Vault"
retry 5 vault_ci_login
export KUBECONFIG=/tmp/kubeconfig
export VALUES_FILE=/tmp/values.yaml

# deploy to dev-asia
vault kv get -field=kubeconfig dev/seed-clusters/dev.kubermatic.io > ${KUBECONFIG}
vault kv get -field=asia-south1-c-values.yaml dev/seed-clusters/dev.kubermatic.io > ${VALUES_FILE}
kubectl config use-context asia-south1-c
echodate "Successfully got secrets for dev-asia from Vault"

echodate "Deploying ${DEPLOY_STACK} stack to dev-asia"
./hack/ci/deploy.sh seed ${VALUES_FILE}
echodate "Successfully deployed ${DEPLOY_STACK} stack to dev-asia"
