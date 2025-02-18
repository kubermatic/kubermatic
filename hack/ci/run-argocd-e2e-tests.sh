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

### This script sets a multi-seed Kubermatic installation in AWS, deploys various applications on it via ArgoCD
### and then runs validation e2e tests using chainsaw

set -euo pipefail
#set -x
cd $(dirname $0)/../..
source hack/lib.sh

EXTRA_ARGS=""
PROVIDER="${PROVIDER:-aws}"
maxDuration=60 # in minutes

if provider_disabled $PROVIDER; then
  exit 0
fi

if [[ $PROVIDER == "aws" ]]; then
  EXTRA_ARGS="-aws-access-key-id=${AWS_E2E_TESTS_KEY_ID}
    -aws-secret-access-key=${AWS_E2E_TESTS_SECRET}
    -aws-kkp-datacenter=aws-eu-west-1a"
fi

# add a bit of setup time to bring up the project, tear it down again etc.
((maxDuration = $maxDuration + 30))

echodate "Running KKP mgmt via ArgoCD CI tests..."

# To upgrade KKP, update the version of kkp here.
KKP_VERSION=v2.27.0-alpha.1
#KKP_VERSION=v2.26.2
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
# echodate "Path:" $PATH

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
  curl -sLO "https://github.com/kubermatic/kubeone/releases/download/v${K1_VERSION}/kubeone_${K1_VERSION}_linux_amd64.zip" &&
    unzip -qq kubeone_${K1_VERSION}_linux_amd64.zip -d kubeone_${K1_VERSION}_linux_amd64 &&
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
  cd -

  if ! [ -x "$(command -v chainsaw)" ]; then
    echodate 'Error: chainsaw testing tool is not installed.' >&2
    exit 1
  fi

  # TODO: Review if we really need to save things once CI starts to work perfectly.
  # download and setup AWS CLI
  curl -s "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
  unzip -q awscliv2.zip
  ./aws/install
  rm -rf awscliv2.zip ./aws
}

function restoreSshKey() {
  echodate "Downloading SSH key pair from s3"
  aws s3 cp s3://kubermatic-e2e-test-tf/kkp-argocd-test/ssh-keys/id_rsa ~/.ssh/id_rsa
  aws s3 cp s3://kubermatic-e2e-test-tf/kkp-argocd-test/ssh-keys/id_rsa.pub ~/.ssh/id_rsa.pub
  chmod 400 ~/.ssh/id_rsa
  eval $(ssh-agent -s) && ssh-add ~/.ssh/id_rsa
}

checkoutTestRepo() {
  echodate "Cloning the argocd gitops Git Repo"
  ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts
  git clone git@github.com:kubermatic-labs/kkp-using-argocd.git
}

createSeedClusters() {
  echodate creating Seed Clusters
  # export TF_LOG=DEBUG
  cd kubeone-install/${MASTER}
  tofu init
  tofu apply -auto-approve
  tofu output -json > tf.json
  ../../../${KUBEONE_INSTALL_DIR}/kubeone apply -t ./tf.json -m kubeone.yaml --auto-approve
  if [ $? -ne 0 ]; then
    echodate kubeone master cluster installation failed.
    exit 2
  fi
  cd ../..
  aws s3 cp ${MASTER_KUBECONFIG} s3://kubermatic-e2e-test-tf/kkp-argocd-test/kubeconfig/

  if [[ ${SEED} != false ]]; then
    cd kubeone-install/${SEED}
    tofu init
    tofu apply -auto-approve
    tofu output -json > tf.json
    ../../../${KUBEONE_INSTALL_DIR}/kubeone apply -t ./tf.json -m kubeone.yaml --auto-approve
    # cd kubeone-install/${SEED} && tofu init && tofu apply -auto-approve
    if [ $? -ne 0 ]; then
      echodate kubeone seed cluster installation failed.
      exit 3
    fi
    cd ../..
    aws s3 cp ${SEED_KUBECONFIG} s3://kubermatic-e2e-test-tf/kkp-argocd-test/kubeconfig/
  fi
}

# Validate kubeone clusters - apiserver availability, smoke test
# TODO: do via chainsaw as well as check apiserver availability
validateSeedClusters() {
  echodate validateSeedClusters: Not implemented.
}

# deploy argo and kkp argo apps
deployArgoApps() {
  echodate Deploying ArgoCD and KKP ArgoCD Apps.
  # TODO: variable for the ingress hostname
  helm repo add dharapvj https://dharapvj.github.io/helm-charts/
  helm repo add argo https://argoproj.github.io/argo-helm
  helm repo update dharapvj
  helm repo update argo
  # master seed
  KUBECONFIG=${MASTER_KUBECONFIG} helm upgrade --install argocd --version ${ARGO_VERSION} --namespace argocd --create-namespace argo/argo-cd -f values-argocd.yaml --set "server.ingress.hosts[0]=argocd.${CLUSTER_PREFIX}.lab.kubermatic.io" --set "server.ingress.tls[0].hosts[0]=argocd.${CLUSTER_PREFIX}.lab.kubermatic.io"
  KUBECONFIG=${MASTER_KUBECONFIG} helm upgrade --install kkp-argo-apps --set kkpVersion=${KKP_VERSION} -f ./${ENV}/demo-master/argoapps-values.yaml dharapvj/argocd-apps

  if [[ ${SEED} != false ]]; then
    KUBECONFIG=${SEED_KUBECONFIG} helm upgrade --install argocd --version ${ARGO_VERSION} --namespace argocd --create-namespace argo/argo-cd -f values-argocd.yaml --set "server.ingress.hosts[0]=argocd.india.${CLUSTER_PREFIX}.lab.kubermatic.io" --set "server.ingress.tls[0].hosts[0]=argocd.india.${CLUSTER_PREFIX}.lab.kubermatic.io"
    KUBECONFIG=${SEED_KUBECONFIG} helm upgrade --install kkp-argo-apps --set kkpVersion=${KKP_VERSION} -f ./${ENV}/india-seed/argoapps-values.yaml dharapvj/argocd-apps
  fi
}
# download kkp release and run kkp installer
installKKP() {
  echodate installing KKP on master seed.
  if [ ! -d "${INSTALL_DIR}" ]; then
    echodate "$INSTALL_DIR does not exist. Downloading KKP release"
    mkdir -p ${INSTALL_DIR}/
    curl -sL "https://github.com/kubermatic/kubermatic/releases/download/${KKP_VERSION}/kubermatic-ee-${KKP_VERSION}-linux-amd64.tar.gz" | tar -xz --directory ${INSTALL_DIR}/
  fi

  # replace imagepullsecret
  export DECODE=$(echo $IMAGE_PULL_SECRET_DATA | base64 -d)
  # set -x
  yq e '.spec.imagePullSecret = strenv(DECODE)' ./${ENV}/demo-master/k8cConfig.yaml > ./${ENV}/demo-master/k8cConfig2.yaml
  # aws s3 cp ./${ENV}/demo-master/k8cConfig2.yaml s3://kubermatic-e2e-test-tf/kkp-argocd-test/kubeconfig/
  # ls -ltr ./${ENV}/demo-master/k8cConfig2.yaml
  # ls -ltr ${MASTER_KUBECONFIG}
  KUBECONFIG=${MASTER_KUBECONFIG} ${INSTALL_DIR}/kubermatic-installer deploy \
    --charts-directory ${INSTALL_DIR}/charts --config ./${ENV}/demo-master/k8cConfig2.yaml --helm-values ./${ENV}/demo-master/values.yaml \
    --skip-charts='cert-manager,nginx-ingress-controller,dex'
  # set +x
}

# generate kubeconfig secret and make a git commit programmatically and push tag
generateNPushSeedKubeConfig() {
  echodate generating and pushing latest Seed Kubeconfig secrets.
  local kubeconfig_b64=$(${INSTALL_DIR}/kubermatic-installer convert-kubeconfig ./kubeone-install/${MASTER}/${CLUSTER_PREFIX}-${MASTER}-kubeconfig | base64 -w0)
  # echodate $kubeconfig_b64
  sed -i "/kubeconfig: /s/: .*/: $(echo $kubeconfig_b64)/" ${ENV}/demo-master/seed-kubeconfig-secret-self.yaml
  # reset
  kubeconfig_b64=""
  if [[ ${SEED} != false ]]; then
    kubeconfig_b64=$(${INSTALL_DIR}/kubermatic-installer convert-kubeconfig ./kubeone-install/${SEED}/${CLUSTER_PREFIX}-${SEED}-kubeconfig | base64 -w0)
    sed -i "/kubeconfig: /s/: .*/: $(echo $kubeconfig_b64)/" ${ENV}/demo-master/seed-kubeconfig-secret-india.yaml
  fi
  # automated git commit and push tag
  git config --global user.email "ci@kubermatic.com"
  git config --global user.name "Kubermatic CI Automation"
  git add ${ENV}/demo-master/seed-kubeconfig-secret-india.yaml ${ENV}/demo-master/seed-kubeconfig-secret-self.yaml
  git commit -m "Adding latest seed kubeconfigs so that Seed resources will reconcile correctly" || echodate "ignore commit failure, proceed"
  git push origin main
  git tag -f ${ENV}-kkp-${KKP_VERSION}
  git push origin -f ${ENV}-kkp-${KKP_VERSION}
}
# TODO: validate installation? Create user clusters, access MLA links etc.
# more the merrier
validateDemoInstallation() {
  echodate Validating the Demo Installation.
  echodate sleeping for many minutes while restarting some services to get cert-manager based certs clearly created.
  # sleep for completion of installation of all services!
  sleep 10m

  # hack: need to work the DNS issues so that certs get created properly
  KUBECONFIG=$PWD/kubeone-install/${MASTER}/argodemo-${MASTER}-kubeconfig kubectl rollout restart sts -n argocd argocd-application-controller
  KUBECONFIG=$PWD/kubeone-install/${MASTER}/argodemo-${MASTER}-kubeconfig kubectl rollout restart deploy -n kube-system coredns
  if [[ ${SEED} != false ]]; then
    KUBECONFIG=$PWD/kubeone-install/${SEED}/argodemo-${SEED}-kubeconfig kubectl rollout restart sts -n argocd argocd-application-controller
    KUBECONFIG=$PWD/kubeone-install/${SEED}/argodemo-${SEED}-kubeconfig kubectl rollout restart deploy -n kube-system coredns
  fi
  sleep 1m
  KUBECONFIG=$PWD/kubeone-install/${MASTER}/argodemo-${MASTER}-kubeconfig kubectl rollout restart ds -n kube-system node-local-dns
  if [[ ${SEED} != false ]]; then
    KUBECONFIG=$PWD/kubeone-install/${SEED}/argodemo-${SEED}-kubeconfig kubectl rollout restart ds -n kube-system node-local-dns
  fi
  sleep 8m
  KUBECONFIG=$PWD/kubeone-install/${MASTER}/argodemo-${MASTER}-kubeconfig kubectl rollout restart deploy -n cert-manager cert-manager
  if [[ ${SEED} != false ]]; then
    KUBECONFIG=$PWD/kubeone-install/${SEED}/argodemo-${SEED}-kubeconfig kubectl rollout restart deploy -n cert-manager cert-manager
  fi
  sleep 6m
  KUBECONFIG=$PWD/kubeone-install/${MASTER}/argodemo-${MASTER}-kubeconfig chainsaw test tests/e2e/master-seed --namespace chainsaw
  if [[ ${SEED} != false ]]; then
    KUBECONFIG=$PWD/kubeone-install/${SEED}/argodemo-${SEED}-kubeconfig chainsaw test tests/e2e/seed-india --namespace chainsaw
  fi
}

# post validation, cleanup
cleanup() {
  echodate cleanup all the cluster resources.
  # first destroy master so that kubermatic-operator is gone otherwise it tries to recreate seed node-port-proxy LB
  KUBECONFIG=${MASTER_KUBECONFIG} kubectl delete app -n argocd nginx-ingress-controller || true
  KUBECONFIG=${MASTER_KUBECONFIG} kubectl delete svc -n nginx-ingress-controller nginx-ingress-controller || true
  KUBECONFIG=${MASTER_KUBECONFIG} kubectl delete svc -n kubermatic nodeport-proxy || true
  cd kubeone-install/${MASTER}
  ../../../${KUBEONE_INSTALL_DIR}/kubeone reset -t ./tf.json -m kubeone.yaml --auto-approve
  tofu init && tofu destroy -auto-approve
  cd ../..

  if [[ ${SEED} != false ]]; then
    # now destroy seed
    KUBECONFIG=${SEED_KUBECONFIG} kubectl delete app -n argocd nginx-ingress-controller || true
    KUBECONFIG=${SEED_KUBECONFIG} kubectl delete svc -n nginx-ingress-controller nginx-ingress-controller || true
    KUBECONFIG=${SEED_KUBECONFIG} kubectl delete svc -n kubermatic nodeport-proxy || true
    cd kubeone-install/${SEED}
    ../../../${KUBEONE_INSTALL_DIR}/kubeone reset -t ./tf.json -m kubeone.yaml --auto-approve
    tofu init && tofu destroy -auto-approve
  fi
}

validatePreReq
restoreSshKey
checkoutTestRepo
cd kkp-using-argocd
# temp
createSeedClusters
validateSeedClusters
deployArgoApps
installKKP
generateNPushSeedKubeConfig
validateDemoInstallation
cleanup

echodate "KKP mgmt via ArgoCD CI tests completed..."
