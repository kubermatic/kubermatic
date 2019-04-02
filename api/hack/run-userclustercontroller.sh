#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -x

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
make user-cluster-controller-manager

KUBECONFIG_USERCLUSTER_CONTROLLER=$(mktemp)
# TODO: append -n cluster-$CLUSTER_NAME here
kubectl get secret admin-kubeconfig -o go-template='{{ index .data "kubeconfig" }}' \
  | base64 -d > ${KUBECONFIG_USERCLUSTER_CONTROLLER}

CA_CERT_USERCLUSTER_CONTROLLER=$(mktemp)
kubectl get secret ca -o json | jq -r '.data["ca.crt"]' | base64 -d > ${CA_CERT_USERCLUSTER_CONTROLLER}

CLUSTER_NAME=$(kubectl get secret admin-kubeconfig -o go-template='{{ .metadata.namespace }}'|sed 's/cluster-//g')
ARGS=""

kubectl get secret admin-kubeconfig -o go-template='{{ .metadata.namespace }}'
if kubectl get cluster ${CLUSTER_NAME} -o yaml |grep openshift -q; then
  ARGS="-openshift=true"
fi

./_build/user-cluster-controller-manager \
    -kubeconfig=$KUBECONFIG_USERCLUSTER_CONTROLLER \
    -logtostderr \
    -metrics-listen-address=127.0.0.1:8087 \
    -health-listen-address=127.0.0.1:8088 \
    -v=4 \
    -namespace=$(kubectl get cluster ${CLUSTER_NAME} -o json | jq -r .status.namespaceName) \
    -openvpn-server-port=$(kubectl get service openvpn-server -o json | jq -r .spec.ports[0].nodePort) \
    -cluster-url=$(kubectl get cluster ${CLUSTER_NAME} -o json | jq -r .address.url) \
    -ca-cert=${CA_CERT_USERCLUSTER_CONTROLLER} \
    ${ARGS}
