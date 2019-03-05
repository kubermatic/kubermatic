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
  | base64 -d > $KUBECONFIG_USERCLUSTER_CONTROLLER

ARGS=""
kubectl get secret admin-kubeconfig -o go-template='{{ .metadata.namespace }}'
if kubectl get cluster $(kubectl get secret admin-kubeconfig -o go-template='{{ .metadata.namespace }}'|sed 's/cluster-//g')\
    -o yaml|grep openshift -q; then
  ARGS="-openshift=true"
fi

./_build/user-cluster-controller-manager \
    -kubeconfig=$KUBECONFIG_USERCLUSTER_CONTROLLER \
    -logtostderr \
    -internal-address=127.0.0.1:8087 \
    -v=4 \
    $ARGS
