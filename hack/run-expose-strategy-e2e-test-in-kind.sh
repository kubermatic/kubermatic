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
### couple of test Presets and Users and then runs the e2e tests for the
### nodeport-proxy.

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

function generate_secret {
  cat /dev/urandom | LC_ALL=C tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1
  echo ''
}

# We replace the domain with a dns name relying on nip.io poining to the
# nodeport-proxy service. This makes the testing of expose strategies relying
# on nodeport-proxy very easy from within the kind cluster.
# TODO(irozzo) Find another solution in case nip.io does not result to be
# available enough for our own CI usage.
function patch_kubermatic_domain {
  local ip="$(kubectl get service nodeport-proxy -n kubermatic -otemplate --template='{{ .spec.clusterIP }}')"
  [ -z "${ip}" ] && return 1
  kubectl patch kubermaticconfigurations.operator.kubermatic.io -n kubermatic e2e \
    --type="json" -p='[{"op": "replace", "path": "/spec/ingress/domain", "value": "'${ip}.nip.io'"}]'
}

DOCKER_REPO="${DOCKER_REPO:-quay.io/kubermatic}"
GOOS="${GOOS:-linux}"
TAG="$(git rev-parse HEAD)"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kubermatic}"
KIND_NODE_VERSION="${KIND_NODE_VERSION:-v1.20.2}"
USER_CLUSTER_KUBERNETES_VERSION="${USER_CLUSTER_KUBERNETES_VERSION:-v1.20.2}"
REPOSUFFIX="${REPOSUFFIX:-""}"
KUBECONFIG="${KUBECONFIG:-"${HOME}/.kube/config"}"
HELM_BINARY="${HELM_BINARY:-helm}" # This works when helm 3 is in path

type kind > /dev/null || fatal \
  "Kind is required to run this script, please refer to: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"

function clean_up {
  echodate "Deleting cluster ${KIND_CLUSTER_NAME}"
  kind delete cluster --name "${KIND_CLUSTER_NAME}" || true
}
appendTrap clean_up EXIT

# Only start docker daemon in CI envorinment.
if [[ ! -z "${JOB_NAME:-}" ]] && [[ ! -z "${PROW_JOB_ID:-}" ]]; then
  start_docker_daemon
fi

# build Docker images
make clean
make docker-build \
  GOOS="${GOOS}" \
  DOCKER_REPO="${DOCKER_REPO}" \
  TAGS="${TAG}"
make -C cmd/nodeport-proxy docker \
  GOOS="${GOOS}" \
  DOCKER_REPO="${DOCKER_REPO}" \
  TAG="${TAG}"
make -C cmd/kubeletdnat-controller docker \
  GOOS="${GOOS}" \
  DOCKER_REPO="${DOCKER_REPO}" \
  TAG="${TAG}"
make -C cmd/user-ssh-keys-agent docker \
  GOOS="${GOOS}" \
  DOCKER_REPO="${DOCKER_REPO}" \
  TAG="${TAG}"
make -C addons docker \
  DOCKER_REPO="${DOCKER_REPO}" \
  TAG="${TAG}"
# the installer should be built for the target platform.
rm _build/kubermatic-installer
make _build/kubermatic-installer

# setup Kind cluster
time retry 5 kind create cluster \
  --name="${KIND_CLUSTER_NAME}" \
  --image=kindest/node:"${KIND_NODE_VERSION}"
kind export kubeconfig --name=${KIND_CLUSTER_NAME}

# load nodeport-proxy image
time retry 5 kind load docker-image "${DOCKER_REPO}/nodeport-proxy:${TAG}" --name "${KIND_CLUSTER_NAME}"
time retry 5 kind load docker-image "${DOCKER_REPO}/addons:${TAG}" --name "${KIND_CLUSTER_NAME}"
time retry 5 kind load docker-image "${DOCKER_REPO}/kubermatic${REPOSUFFIX}:${TAG}" --name "${KIND_CLUSTER_NAME}"
time retry 5 kind load docker-image "${DOCKER_REPO}/kubeletdnat-controller${REPOSUFFIX}:${TAG}" --name "${KIND_CLUSTER_NAME}"
time retry 5 kind load docker-image "${DOCKER_REPO}/user-ssh-keys-agent:${TAG}" --name "${KIND_CLUSTER_NAME}"

# This is just used as a const
# NB: The CE requires Seeds to be named this way
export SEED_NAME=kubermatic

# Tell the conformance tester what dummy account we configure for the e2e tests.
export KUBERMATIC_OIDC_LOGIN="roxy@loodse.com"
export KUBERMATIC_OIDC_PASSWORD="password"

# Build binaries and load the Docker images into the kind cluster
echodate "Building binaries for ${TAG}"
TEST_NAME="Build Kubermatic binaries"

echodate "Successfully built and loaded all images"

TMPDIR="$(mktemp -d -t k8c-XXXXXXXXXX)"
echo "Config dir: ${TMPDIR}"
# prepare to run kubermatic-installer
KUBERMATIC_CONFIG="${TMPDIR}/kubermatic.yaml"

cat << EOF > ${KUBERMATIC_CONFIG}
apiVersion: operator.kubermatic.io/v1alpha1
kind: KubermaticConfiguration
metadata:
  name: e2e
  namespace: kubermatic
spec:
  ingress:
    domain: 127.0.0.1.nip.io
    disable: true
  userCluster:
    apiserverReplicas: 1
  api:
    replicas: 0
    debugLog: true
  featureGates:
    TunnelingExposeStrategy: {}
  ui:
    replicas: 0
  # Dex integration
  auth:
    #tokenIssuer: "http://dex.oauth:5556/dex"
    #issuerRedirectURL: "http://localhost:8000"
    tokenIssuer: "https://127.0.0.1.nip.io/dex"
    serviceAccountKey: "$(generate_secret)"
EOF

HELM_VALUES_FILE="${TMPDIR}/values.yaml"
cat << EOF > ${HELM_VALUES_FILE}
dex:
  replicas: 0
kubermaticOperator:
  image:
    repository: "quay.io/kubermatic/kubermatic${REPOSUFFIX}"
    tag: "${TAG}"
EOF

# install dependencies and Kubermatic Operator into cluster
./_build/kubermatic-installer deploy \
  --storageclass copy-default \
  --config "${KUBERMATIC_CONFIG}" \
  --kubeconfig "${KUBECONFIG}" \
  --helm-values "${HELM_VALUES_FILE}" \
  --helm-binary "${HELM_BINARY}"

# TODO: The installer should wait for everything to finish reconciling.
#echodate "Waiting for Kubermatic Operator to deploy Master components..."
# sleep a bit to prevent us from checking the Deployments too early, before
# the operator had time to reconcile
sleep 5
retry 10 check_all_deployments_ready kubermatic

echodate "Finished installing Kubermatic"

echodate "Installing Seed..."
SEED_MANIFEST="${TMPDIR}/seed.yaml"

SEED_KUBECONFIG="$(cat ${KUBECONFIG} | sed 's/127.0.0.1.*/kubernetes.default.svc.cluster.local./' | base64 -w0)"

cat << EOF > ${SEED_MANIFEST}
kind: Secret
apiVersion: v1
metadata:
  name: "${SEED_NAME}-kubeconfig"
  namespace: kubermatic
data:
  kubeconfig: "${SEED_KUBECONFIG}"

---
kind: Seed
apiVersion: kubermatic.k8s.io/v1
metadata:
  name: "${SEED_NAME}"
  namespace: kubermatic
  labels:
    worker-name: ""
spec:
  country: Germany
  location: Hamburg
  kubeconfig:
    name: "${SEED_NAME}-kubeconfig"
    namespace: kubermatic
    fieldPath: kubeconfig
  datacenters:
    byo-kubernetes:
      location: Frankfurt
      country: DE
      spec:
        bringyourown: {}
  expose_strategy: Tunneling
EOF

retry 3 kubectl apply -f $SEED_MANIFEST
echodate "Finished installing Seed"

sleep 5
echodate "Waiting for Kubermatic Operator to deploy Seed components..."
retry 8 check_all_deployments_ready kubermatic
echodate "Kubermatic Seed is ready."

echodate "Patching Kubermatic ingress domain with nodeport-proxy service cluster IP..."
retry 5 patch_kubermatic_domain
echodate "Kubermatic ingress domain patched."

# run tests
# use ginkgo binary by preference to have better output:
# https://github.com/onsi/ginkgo/issues/633
if type ginkgo > /dev/null; then
  ginkgo --tags=e2e -v pkg/test/e2e/expose-strategy/ \
    -r \
    --randomizeAllSpecs \
    --randomizeSuites \
    --failOnPending \
    --cover \
    --trace \
    --race \
    --progress \
    -- --kubeconfig "${HOME}/.kube/config" \
    -- --kubeconfig "${HOME}/.kube/config" \
    --kubernetes-version "${USER_CLUSTER_KUBERNETES_VERSION}" \
    --datacenter byo-kubernetes \
    --debug-log
else
  CGO_ENABLED=1 go test --tags=e2e -v -race ./pkg/test/e2e/expose-strategy/... \
    --ginkgo.randomizeAllSpecs \
    --ginkgo.failOnPending \
    --ginkgo.trace \
    --ginkgo.progress \
    --kubeconfig "${HOME}/.kube/config" \
    --kubernetes-version "${USER_CLUSTER_KUBERNETES_VERSION}" \
    --datacenter byo-kubernetes \
    --debug-log
fi
