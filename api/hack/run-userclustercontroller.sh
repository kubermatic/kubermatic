#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
make user-cluster-controller-manager

# Getting everything we need from the api
# TODO: append -n cluster-$CLUSTER_NAME here or switch to the namespace before
ADMIN_KUBECONFIG_RAW="$(kubectl get secret admin-kubeconfig -o json)"
OPENVPN_CA_SECRET_RAW="$(kubectl get secret openvpn-ca -o json)"
CLUSTER_RAW="$(kubectl get cluster $(echo $ADMIN_KUBECONFIG_RAW|jq -r '.metadata.namespace'|sed 's/cluster-//') -o json)"
OPENVPN_SERVER_SERVICE_RAW="$(kubectl get service openvpn-server -o json )"

CA_CERT_USERCLUSTER_CONTROLLER_FILE=$(mktemp)
OPENVPN_CA_CERT_FILE=$(mktemp)
OPENVPN_CA_KEY_FILE=$(mktemp)
KUBECONFIG_USERCLUSTER_CONTROLLER_FILE=$(mktemp)

kubectl get secret ca -o json | jq -r '.data["ca.crt"]' | base64 -d > ${CA_CERT_USERCLUSTER_CONTROLLER_FILE}
echo ${OPENVPN_CA_SECRET_RAW}|jq -r '.data["ca.crt"]'|base64 -d > ${OPENVPN_CA_CERT_FILE}
echo ${OPENVPN_CA_SECRET_RAW}|jq -r '.data["ca.key"]'|base64 -d > ${OPENVPN_CA_KEY_FILE}
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
    -openvpn-ca-cert-file=${OPENVPN_CA_CERT_FILE} \
    -openvpn-ca-key-file=${OPENVPN_CA_KEY_FILE} \
    -cluster-url=${CLUSTER_URL} \
    -ca-cert=${CA_CERT_USERCLUSTER_CONTROLLER_FILE} \
    -version=${CLUSTER_VERSION} \
    -log-debug=true \
    -log-format=Console \
    ${ARGS}
