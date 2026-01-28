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

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

KUBERMATIC_DEBUG=${KUBERMATIC_DEBUG:-true}
PPROF_PORT=${PPROF_PORT:-6601}
TMPDIR="${TMPDIR:-$(mktemp -d)}"

echodate "Compiling user-cluster-controller-manager..."
make user-cluster-controller-manager

# Getting everything we need from the api
# This script assumes you are in your cluster namespace, which you can configure via `kubectl config set-context --current --namespace=<<cluster-namespace>>`
NAMESPACE="${NAMESPACE:-$(kubectl config view --minify | grep namespace | awk '{ print $2 }')}"
CLUSTER_NAME="$(echo $NAMESPACE | sed 's/cluster-//')"
CLUSTER_RAW="$(kubectl get cluster $CLUSTER_NAME -o json)"
CLUSTER_URL="$(echo $CLUSTER_RAW | jq -r .status.address.url)"

if [ -z "${OWNER_EMAIL:-}" ]; then
  echo "You must set the email address of the cluster owner \"\$OWNER_EMAIL\", otherwise the controller will fail to start"
  exit 1
fi

# We cannot use the `admin-kubeconfig` secret because the user-cluster-controller-manager is
# the one creating it in case of openshift. So we just use the internal kubeconfig and replace
# the apiserver url
KUBECONFIG_USERCLUSTER_CONTROLLER_FILE=$(mktemp)
kubectl --namespace "$NAMESPACE" get secret admin-kubeconfig --output json |
  jq '.data.kubeconfig' -r |
  base64 -d \
    > $KUBECONFIG_USERCLUSTER_CONTROLLER_FILE
echo "Using kubeconfig $KUBECONFIG_USERCLUSTER_CONTROLLER_FILE"

SEED_KUBECONFIG=$(mktemp)
SEED_SERVICEACCOUNT_TOKEN="$(kubectl --namespace "$NAMESPACE" create token kubermatic-usercluster-controller-manager --duration=8h)"
kubectl config view --flatten --minify -ojson |
  jq --arg token "$SEED_SERVICEACCOUNT_TOKEN" 'del(.users[0].user)|.users[0].user.token = $token' > $SEED_KUBECONFIG

CLUSTER_VERSION="$(echo $CLUSTER_RAW | jq -r '.spec.version')"

ARGS=""
if echo $CLUSTER_RAW | grep -i aws -q; then
  ARGS="$ARGS -cloud-provider-name=aws"
elif echo $CLUSTER_RAW | grep -i vsphere -q; then
  ARGS="$ARGS -cloud-provider-name=vsphere"
fi

if echo $CLUSTER_RAW | grep -i kubevirt -q; then
  KUBEVIRT_INFRA_KUBECONFIG=$(mktemp)
  kubectl --namespace "$NAMESPACE" get secret kubevirt-infra-kubeconfig --output json |
    jq '.data."infra-kubeconfig"' -r |
    base64 -d \
      > $KUBEVIRT_INFRA_KUBECONFIG
  echo "Using kubevirt infra kubeconfig $KUBEVIRT_INFRA_KUBECONFIG"
  ARGS="$ARGS -kv-vmi-eviction-controller"
  ARGS="$ARGS -kv-infra-kubeconfig=${KUBEVIRT_INFRA_KUBECONFIG}"
fi

if $(echo ${CLUSTER_RAW} | jq -r '.spec.clusterNetwork.konnectivityEnabled'); then
  KONNECTIVITY_SERVER_SERVICE_RAW="$(kubectl --namespace "$NAMESPACE" get service konnectivity-server -o json)"
  if $(echo ${KONNECTIVITY_SERVER_SERVICE_RAW} | jq --exit-status '.spec.ports[0].nodePort' > /dev/null); then
    KONNECTIVITY_SERVER_PORT="$(echo ${KONNECTIVITY_SERVER_SERVICE_RAW} | jq -r '.spec.ports[0].nodePort')"
    KONNECTIVITY_SERVER_HOST="$(echo ${CLUSTER_RAW} | jq -r '.status.address.externalName')"
  else
    KONNECTIVITY_SERVER_PORT="$(echo ${CLUSTER_RAW} | jq -r '.status.address.port')"
    KONNECTIVITY_SERVER_HOST="konnectivity-server.$(echo ${CLUSTER_RAW} | jq -r '.status.address.externalName')"
    ARGS="$ARGS -tunneling-agent-ip=100.64.30.10"
  fi
  ARGS="$ARGS -konnectivity-enabled=true"
  ARGS="$ARGS -konnectivity-server-host=${KONNECTIVITY_SERVER_HOST}"
  ARGS="$ARGS -konnectivity-server-port=${KONNECTIVITY_SERVER_PORT}"
fi

APPTMPDIR="$(mktemp -d ${TMPDIR}/application.XXXXX)"

echodate "Starting user-cluster-controller-manager..."
set -x
./_build/user-cluster-controller-manager \
  -kubeconfig=${KUBECONFIG_USERCLUSTER_CONTROLLER_FILE} \
  -ca-bundle=charts/kubermatic-operator/static/ca-bundle.pem \
  -metrics-listen-address=127.0.0.1:8087 \
  -health-listen-address=127.0.0.1:8088 \
  -pprof-listen-address=":${PPROF_PORT}" \
  -namespace=${NAMESPACE} \
  -cluster-name=${CLUSTER_NAME} \
  -cluster-url=${CLUSTER_URL} \
  -version=${CLUSTER_VERSION} \
  -log-debug=$KUBERMATIC_DEBUG \
  -log-format=Console \
  -logtostderr \
  -v=4 \
  -seed-kubeconfig=${SEED_KUBECONFIG} \
  -owner-email=${OWNER_EMAIL} \
  -dns-cluster-ip=10.240.16.10 \
  -application-cache="${APPTMPDIR}" \
  ${ARGS}
