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

### This script is used as a postsubmit job and updates the dev master
### cluster after every commit to main.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

export DEPLOY_STACK=${DEPLOY_STACK:-kubermatic}
export GIT_HEAD_HASH="$(git rev-parse HEAD | tr -d '\n')"
export VAULT_VALUES_FIELD=helm-master.yaml

# per-stack customizations
case ${DEPLOY_STACK} in
kubermatic)
  ./hack/ci/release-images.sh
  ;;

usercluster-mla)
  export VAULT_VALUES_FIELD=helm-seed-shared-usercluster-mla.yaml
  NO_IMAGES=true BINARY_NAMES="kubermatic-installer" ./hack/ci/release-images.sh
  ;;

seed-mla)
  export VAULT_VALUES_FIELD=helm-seed-shared-mla.yaml
  NO_IMAGES=true BINARY_NAMES="kubermatic-installer" ./hack/ci/release-images.sh
  ;;

esac

echodate "Getting secrets from Vault"
retry 5 vault_ci_login
export KUBECONFIG=/tmp/kubeconfig
export VALUES_FILE=/tmp/values.yaml
export IMAGE_PULL_SECRET=/tmp/dockercfg
export KUBERMATIC_CONFIG=/tmp/kubermatic.yaml

vault kv get -field=kubeconfig dev/seed-clusters/dev.kubermatic.io > ${KUBECONFIG}
vault kv get -field=${VAULT_VALUES_FIELD} dev/seed-clusters/dev.kubermatic.io > ${VALUES_FILE}
vault kv get -field=.dockerconfigjson dev/seed-clusters/dev.kubermatic.io > ${IMAGE_PULL_SECRET}
vault kv get -field=kubermatic.yaml dev/seed-clusters/dev.kubermatic.io > ${KUBERMATIC_CONFIG}
echodate "Successfully got secrets for dev from Vault"

echodate "Deploying ${DEPLOY_STACK} stack to dev"
./hack/ci/deploy.sh master ${VALUES_FILE}
echodate "Successfully deployed ${DEPLOY_STACK} stack to dev"
