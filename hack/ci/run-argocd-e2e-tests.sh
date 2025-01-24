#!/usr/bin/env bash

# Copyright 2025 The Kubermatic Kubernetes Platform contributors.
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
### couple of test Presets and Users and then runs the IPAM e2e tests.

set -euo pipefail
#set -x
cd $(dirname $0)/../..
source hack/lib.sh

if [ -z "${E2E_SSH_PUBKEY:-}" ]; then
  echodate "Getting default SSH pubkey for machines from Vault"
  retry 5 vault_ci_login
  E2E_SSH_PUBKEY="$(mktemp)"
  vault kv get -field=pubkey dev/e2e-machine-controller-ssh-key > "${E2E_SSH_PUBKEY}"
else
  E2E_SSH_PUBKEY_CONTENT="${E2E_SSH_PUBKEY}"
  E2E_SSH_PUBKEY="$(mktemp)"
  echodate "${E2E_SSH_PUBKEY_CONTENT}" > "${E2E_SSH_PUBKEY}"
fi

echodate "SSH public key will be $(head -c 25 ${E2E_SSH_PUBKEY})...$(tail -c 25 ${E2E_SSH_PUBKEY})"

EXTRA_ARGS=""
provider="${PROVIDER:-aws}"
maxDuration=60 # in minutes

if provider_disabled $provider; then
  exit 0
fi

if [[ $provider == "aws" ]]; then
  EXTRA_ARGS="-aws-access-key-id=${AWS_E2E_TESTS_KEY_ID}
    -aws-secret-access-key=${AWS_E2E_TESTS_SECRET}
    -aws-kkp-datacenter=aws-eu-west-1a"
fi

# add a bit of setup time to bring up the project, tear it down again etc.
((maxDuration = $maxDuration + 30))

echodate "Running KKP mgmt via ArgoCD CI tests..."

# To upgrade KKP, update the version of kkp here.
#KKP_VERSION=v2.25.11
KKP_VERSION=v2.26.2

K1_VERSION=1.8.3
ARGO_VERSION=5.36.10
CHAINSAW_VERSION=0.2.12
ENV=dev
MASTER=dev-master
# SEED=false # - don't create extra seed. Any other value - name of the seed
SEED=dev-seed
CLUSTER_PREFIX=argodemo

INSTALL_DIR=./binaries/kubermatic/releases/${KKP_VERSION}
KUBEONE_INSTALL_DIR=./binaries/kubeone/releases/${K1_VERSION}
MASTER_KUBECONFIG=./kubeone-install/${MASTER}/${CLUSTER_PREFIX}-${MASTER}-kubeconfig
SEED_KUBECONFIG=./kubeone-install/${SEED}/${CLUSTER_PREFIX}-${SEED}-kubeconfig
export AWS_ACCESS_KEY_ID=${AWS_E2E_TESTS_KEY_ID}
export AWS_SECRET_ACCESS_KEY=${AWS_E2E_TESTS_SECRET}
echodate "Path:" $PATH

# LOGIC
# validate that we have kubeone, kubectl, helm, git, sed, chainsaw binaries available
# TODO: validate availability of ssh-agent?
validatePreReq() {
  echodate validate Prerequisites.
  if [[ -n "${AWS_ACCESS_KEY_ID-}" && -n "${AWS_SECRET_ACCESS_KEY-}" ]]; then
    echodate AWS credentials found! Proceeding.
  elif [[ -n "${AWS_PROFILE-}" ]]; then
    echodate AWS profile variable found! Proceeding.
  else
    echodate No AWS credentials configured. You must export either combination of AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY OR AWS_PROFILE env variable. Exiting the script.
    exit 1
  fi

  if ! [ -x "$(command -v git)" ]; then
    echodate 'Error: git is not installed.' >&2
    exit 1
  fi

	mkdir -p ${KUBEONE_INSTALL_DIR}
	curl -sLO "https://github.com/kubermatic/kubeone/releases/download/v${K1_VERSION}/kubeone_${K1_VERSION}_linux_amd64.zip" && \
    unzip -qq kubeone_${K1_VERSION}_linux_amd64.zip -d kubeone_${K1_VERSION}_linux_amd64 && \
    mv kubeone_${K1_VERSION}_linux_amd64/kubeone ${KUBEONE_INSTALL_DIR} && rm -rf kubeone_${K1_VERSION}_linux_amd64 kubeone_${K1_VERSION}_linux_amd64.zip

  if ! [ -x ${KUBEONE_INSTALL_DIR}/kubeone ]; then
    echodate 'Error: kubeone is not installed.' >&2
    exit 1
  fi

  if ! [ -x "$(command -v helm)" ]; then
    echodate 'Error: helm is not installed.' >&2
    exit 1
  fi

  if ! [ -x "$(command -v sed)" ]; then
    echodate 'Error: sed is not installed.' >&2
    exit 1
  fi

  if ! [ -x "$(command -v tofu)" ]; then
    echodate 'Error: tofu is not installed.' >&2
    exit 1
  fi

  cd /tmp
  curl -sL https://github.com/kyverno/chainsaw/releases/download/v${CHAINSAW_VERSION}/chainsaw_linux_amd64.tar.gz | tar -xz
  mv chainsaw /usr/local/bin

  if ! [ -x "$(command -v chainsaw)" ]; then
    echodate 'Error: chainsaw testing tool is not installed.' >&2
    exit 1
  fi
}

checkoutTestRepo() {
  git clone https://github.com/kubermatic-labs/kkp-using-argocd.git
}

createSeedClusters(){ 
  echo creating Seed Clusters
#  cd kubeone-install/${MASTER} && tofu init && tofu apply -auto-approve &&../../${KUBEONE_INSTALL_DIR}/kubeone apply -t . -m kubeone.yaml --auto-approve
  # export TF_LOG=DEBUG
  cd kubeone-install/${MASTER} && tofu init && tofu apply -auto-approve
  if [ $? -ne 0 ]; then
    echo kubeone master cluster installation failed.
    exit 2
  fi
  cd ../..

  # if [[ ${SEED} != false ]]; then
  #   # cd kubeone-install/${SEED} && tofu init && tofu apply -auto-approve &&../../${KUBEONE_INSTALL_DIR}/kubeone apply -t . -m kubeone.yaml --auto-approve
  #   cd kubeone-install/${SEED} && tofu init && tofu plan
  #   if [ $? -ne 0 ]; then
  #     echo kubeone seed cluster installation failed.
  #     exit 3
  #   fi
  #   cd ../..
  # fi
}

validatePreReq
checkoutTestRepo
cd kkp-using-argocd
createSeedClusters

echodate "KKP mgmt via ArgoCD CI tests completed..."
