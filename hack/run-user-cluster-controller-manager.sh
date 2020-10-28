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

echodate "Compiling user-cluster-controller-manager..."
make user-cluster-controller-manager

# Getting everything we need from the api
# This script assumes you are in your cluster namespace, which you can configure via `kubectl config set-context --current --namespace=<<cluster-namespace>>`
NAMESPACE="${NAMESPACE:-$(kubectl config view --minify | grep namespace |awk '{ print $2 }')}"
CLUSTER_NAME="$(echo $NAMESPACE | sed 's/cluster-//')"
CLUSTER_RAW="$(kubectl get cluster $CLUSTER_NAME -o json)"
CLUSTER_URL="$(echo $CLUSTER_RAW | jq -r '.address.url')"

if [ -z "${OWNER_EMAIL:-}" ]; then
  echo "You must set the email address of the cluster owner \"\$OWNER_EMAIL\", otherwise the controller will fail to start"
  exit 1
fi

# We can not use the `admin-kubeconfig` secret because the user-cluster-controller-manager is
# the one creating it in case of openshift. So we just use the internal kubeconfig and replace
# the apiserver uurl
KUBECONFIG_USERCLUSTER_CONTROLLER_FILE=$(mktemp)
ADMIN_KUBECONFIG="$(kubectl get secret internal-admin-kubeconfig -o json \
  | jq '.data.kubeconfig' -r \
  | base64 -d \
  | yaml2json \
  | jq --arg url "$CLUSTER_URL" '.clusters[0].cluster.server = $url')"
echo $ADMIN_KUBECONFIG > $KUBECONFIG_USERCLUSTER_CONTROLLER_FILE
echo "Using kubeconfig $KUBECONFIG_USERCLUSTER_CONTROLLER_FILE"

OPENVPN_SERVER_SERVICE_RAW="$(kubectl get service openvpn-server -o json)"

SEED_SERVICEACCOUNT_TOKEN="$(kubectl get secret -o json \
  | jq '.items[]|select(.metadata.annotations["kubernetes.io/service-account.name"] == "kubermatic-usercluster-controller-manager")|.data.token' -r \
  | base64 -d)"
SEED_KUBECONFIG=$(mktemp)
kubectl config view  --flatten --minify -ojson \
  | jq --arg token "$SEED_SERVICEACCOUNT_TOKEN" 'del(.users[0].user)|.users[0].user.token = $token' > $SEED_KUBECONFIG

CLUSTER_VERSION="$(echo $CLUSTER_RAW|jq -r '.spec.version')"
CLUSTER_URL="$(echo $CLUSTER_RAW | jq -r .address.url)"
OPENVPN_SERVER_NODEPORT="$(echo ${OPENVPN_SERVER_SERVICE_RAW} | jq -r .spec.ports[0].nodePort)"
CONSOLE_CALLBACK_URI="$(echo $CLUSTER_RAW | jq -r '.address.openshiftConsoleCallback')"

ARGS=""
if echo $CLUSTER_RAW | grep openshift -q; then
  ARGS="-openshift=true"
fi

if echo $CLUSTER_RAW | grep -i aws -q; then
  ARGS="$ARGS -cloud-provider-name=aws"
fi

echodate "Starting user-cluster-controller-manager..."
set -x
./_build/user-cluster-controller-manager \
  -kubeconfig=${KUBECONFIG_USERCLUSTER_CONTROLLER_FILE} \
  -metrics-listen-address=127.0.0.1:8087 \
  -health-listen-address=127.0.0.1:8088 \
  -namespace=${NAMESPACE} \
  -openvpn-server-port=${OPENVPN_SERVER_NODEPORT} \
  -cluster-url=${CLUSTER_URL} \
  -version=${CLUSTER_VERSION} \
  -openshift-console-callback-uri="${CONSOLE_CALLBACK_URI}" \
  -log-debug=$KUBERMATIC_DEBUG \
  -log-format=Console \
  -logtostderr \
  -v=4 \
  -seed-kubeconfig=${SEED_KUBECONFIG} \
  -owner-email=${OWNER_EMAIL} \
  -dns-cluster-ip=10.240.16.10 \
  ${ARGS}
