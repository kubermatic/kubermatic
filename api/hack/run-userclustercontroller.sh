#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
make user-cluster-controller-manager

# Getting everything we need from the api
# TODO: append -n cluster-$CLUSTER_NAME here or switch to the namespace before
ADMIN_KUBECONFIG_RAW="$(kubectl get secret admin-kubeconfig -o json)"
CLUSTER_RAW="$(kubectl get cluster $(echo $ADMIN_KUBECONFIG_RAW|jq -r '.metadata.namespace'|sed 's/cluster-//') -o json)"
OPENVPN_SERVER_SERVICE_RAW="$(kubectl get service openvpn-server -o json )"

SEED_SERVICEACCOUNT_TOKEN="$(kubectl get secret -o json \
  |jq '.items[]|select(.metadata.annotations["kubernetes.io/service-account.name"] == "kubermatic-usercluster-controller-manager")|.data.token' -r \
  |base64 -d)"
SEED_KUBECONFIG=$(mktemp)
kubectl config view  --flatten --minify -ojson \
  |jq --arg token "$SEED_SERVICEACCOUNT_TOKEN" 'del(.users[0].user)|.users[0].user.token = $token' > $SEED_KUBECONFIG

KUBECONFIG_USERCLUSTER_CONTROLLER_FILE=$(mktemp)

echo ${ADMIN_KUBECONFIG_RAW}|jq -r '.data.kubeconfig' |base64 -d > ${KUBECONFIG_USERCLUSTER_CONTROLLER_FILE}

CLUSTER_VERSION="$(echo $CLUSTER_RAW|jq -r '.spec.version')"
CLUSTER_NAMESPACE="$(echo $ADMIN_KUBECONFIG_RAW|jq -r '.metadata.namespace')"
CLUSTER_URL="$(echo $CLUSTER_RAW | jq -r .address.url)"
OPENVPN_SERVER_NODEPORT="$(echo ${OPENVPN_SERVER_SERVICE_RAW} | jq -r .spec.ports[0].nodePort)"

ARGS=""
if echo $CLUSTER_RAW |grep openshift -q; then
  ARGS="-openshift=true"
fi

if echo $CLUSTER_RAW|grep -i aws -q; then
  ARGS="$ARGS -cloud-provider-name=aws"
fi

./_build/user-cluster-controller-manager \
    -kubeconfig=${KUBECONFIG_USERCLUSTER_CONTROLLER_FILE} \
    -metrics-listen-address=127.0.0.1:8087 \
    -health-listen-address=127.0.0.1:8088 \
    -namespace=${CLUSTER_NAMESPACE} \
    -openvpn-server-port=${OPENVPN_SERVER_NODEPORT} \
    -cluster-url=${CLUSTER_URL} \
    -version=${CLUSTER_VERSION} \
    -log-debug=true \
    -log-format=Console \
    -logtostderr \
    -v=4 \
    -seed-kubeconfig=${SEED_KUBECONFIG} \
    ${ARGS}
