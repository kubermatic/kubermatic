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

# We replace the domain with a dns name relying on nip.io pointing to the
# nodeport-proxy service. This makes the testing of expose strategies relying
# on nodeport-proxy very easy from within the kind cluster.
# TODO Find another solution in case nip.io does not result to be
# available enough for our own CI usage.
function patch_kubermatic_domain {
  local ip="$(kubectl get service nodeport-proxy -n kubermatic -otemplate --template='{{ .spec.clusterIP }}')"
  [ -z "${ip}" ] && return 1
  kubectl patch kubermaticconfigurations.kubermatic.k8c.io -n kubermatic e2e \
    --type="json" -p='[{"op": "replace", "path": "/spec/ingress/domain", "value": "'${ip}.nip.io'"}]'
}

DOCKER_REPO="${DOCKER_REPO:-quay.io/kubermatic}"
GOOS="${GOOS:-linux}"
TAG="$(git rev-parse HEAD)"
FAKE_TAG="v0.0.0-test"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kubermatic}"
KUBECONFIG="${KUBECONFIG:-"${HOME}/.kube/config"}"
KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ce}"

REPOSUFFIX=""
if [ "$KUBERMATIC_EDITION" == "ee" ]; then
  REPOSUFFIX="-$KUBERMATIC_EDITION"
fi

type kind > /dev/null || fatal \
  "Kind is required to run this script, please refer to: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"

function clean_up {
  echodate "Deleting cluster ${KIND_CLUSTER_NAME}"
  kind delete cluster --name "${KIND_CLUSTER_NAME}" || true
}
appendTrap clean_up EXIT

# Only start docker daemon in CI envorinment.
if [[ ! -z "${JOB_NAME:-}" ]] && [[ ! -z "${PROW_JOB_ID:-}" ]]; then
  start_docker_daemon_ci
  make download-gocache

  echodate "Preloading the kindest/node image"
  docker load --input /kindest.tar
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
make -C cmd/etcd-launcher docker \
  GOOS="${GOOS}" \
  DOCKER_REPO="${DOCKER_REPO}" \
  TAG="${TAG}"
make -C addons docker \
  DOCKER_REPO="${DOCKER_REPO}" \
  TAG="${TAG}"
make -C cmd/network-interface-manager docker \
  GOOS="${GOOS}" \
  DOCKER_REPO="${DOCKER_REPO}" \
  TAG="${TAG}"
# image with FAKE TAG v0.0.0-test is used by envoy-agent pod in ExposeStrategyTunneling test case
make -C cmd/network-interface-manager docker \
  GOOS="${GOOS}" \
  DOCKER_REPO="${DOCKER_REPO}" \
  TAG="${FAKE_TAG}"

# the installer should be built for the target platform.
rm _build/kubermatic-installer
make _build/kubermatic-installer

# setup Kind cluster
time kind create cluster --name="${KIND_CLUSTER_NAME}"
kind export kubeconfig --name=${KIND_CLUSTER_NAME}

# load nodeport-proxy image
time kind load docker-image "${DOCKER_REPO}/etcd-launcher:${TAG}" --name "${KIND_CLUSTER_NAME}"
time kind load docker-image "${DOCKER_REPO}/nodeport-proxy:${TAG}" --name "${KIND_CLUSTER_NAME}"
time kind load docker-image "${DOCKER_REPO}/addons:${TAG}" --name "${KIND_CLUSTER_NAME}"
time kind load docker-image "${DOCKER_REPO}/kubermatic${REPOSUFFIX}:${TAG}" --name "${KIND_CLUSTER_NAME}"
time kind load docker-image "${DOCKER_REPO}/network-interface-manager:${TAG}" --name "${KIND_CLUSTER_NAME}"
# load network-interface-manager with FAKE TAG 'v0.0.0-test' which is used in ExposeStrategyTunneling test case
time kind load docker-image "${DOCKER_REPO}/network-interface-manager:${FAKE_TAG}" --name "${KIND_CLUSTER_NAME}"

# This is just used as a const
# NB: The CE requires Seeds to be named this way
export SEED_NAME=kubermatic

# Build binaries and load the Docker images into the kind cluster
echodate "Building binaries for ${TAG}"
TEST_NAME="Build Kubermatic binaries"

echodate "Successfully built and loaded all images"

TMPDIR="$(mktemp -d -t k8c-XXXXXXXXXX)"
echo "Config dir: ${TMPDIR}"
# prepare to run kubermatic-installer
KUBERMATIC_CONFIG="${TMPDIR}/kubermatic.yaml"

cat << EOF > ${KUBERMATIC_CONFIG}
apiVersion: kubermatic.k8c.io/v1
kind: KubermaticConfiguration
metadata:
  name: e2e
  namespace: kubermatic
spec:
  ingress:
    domain: 127.0.0.1.nip.io
    disable: true
  featureGates:
    HeadlessInstallation: true
EOF

HELM_VALUES_FILE="${TMPDIR}/values.yaml"
cat << EOF > ${HELM_VALUES_FILE}
kubermaticOperator:
  image:
    repository: "quay.io/kubermatic/kubermatic${REPOSUFFIX}"
    tag: "${TAG}"
EOF

# prepare CRDs
copy_crds_to_chart
set_crds_version_annotation

# gather logs
if [ -n "${ARTIFACTS:-}" ] && [ -x "$(command -v protokol)" ]; then
  # gather the logs of all things in the cluster control plane and in the Kubermatic namespace
  protokol --kubeconfig "${HOME}/.kube/config" --flat --output "$ARTIFACTS/logs/cluster-control-plane" --namespace 'cluster-*' > /dev/null 2>&1 &
  protokol --kubeconfig "${HOME}/.kube/config" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &
fi

# install dependencies and Kubermatic Operator into cluster
./_build/kubermatic-installer deploy --disable-telemetry \
  --storageclass copy-default \
  --config "${KUBERMATIC_CONFIG}" \
  --kubeconfig "${KUBECONFIG}" \
  --helm-values "${HELM_VALUES_FILE}"

# TODO: The installer should wait for everything to finish reconciling.
echodate "Waiting for Kubermatic Operator to deploy Master components..."
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
apiVersion: kubermatic.k8c.io/v1
metadata:
  name: "${SEED_NAME}"
  namespace: kubermatic
spec:
  country: Germany
  location: Hamburg
  kubeconfig:
    name: "${SEED_NAME}-kubeconfig"
    fieldPath: kubeconfig
  datacenters:
    byo-kubernetes:
      location: Frankfurt
      country: DE
      spec:
        bringyourown: {}
  exposeStrategy: Tunneling
EOF

retry 3 kubectl apply --filename $SEED_MANIFEST
retry 5 check_seed_ready kubermatic "$SEED_NAME"
echodate "Finished installing Seed"

sleep 5
echodate "Waiting for Deployments to roll out..."
retry 9 check_all_deployments_ready kubermatic
echodate "Kubermatic is ready."

echodate "Patching Kubermatic ingress domain with nodeport-proxy service cluster IP..."
retry 5 patch_kubermatic_domain
echodate "Kubermatic ingress domain patched."

echodate "Running tests..."

go_test expose_strategy_e2e -tags "$KUBERMATIC_EDITION,e2e" -v ./pkg/test/e2e/expose-strategy \
  -cluster-version "${USER_CLUSTER_KUBERNETES_VERSION:-}" \
  -byo-kkp-datacenter byo-kubernetes

echodate "Done."
