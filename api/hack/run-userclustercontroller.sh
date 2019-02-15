#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -x

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
make usercluster-controller-manager

KUBECONFIG_USERCLUSTER_CONTROLLER=$(mktemp)
# TODO: append -n cluster-$CLUSTER_NAME here
kubectl get secret admin-kubeconfig -o go-template='{{ index .data "kubeconfig" }}' \
  | base64 -d > $KUBECONFIG_USERCLUSTER_CONTROLLER

./_build/usercluster-controller-manager \
    -kubeconfig=$KUBECONFIG_USERCLUSTER_CONTROLLER
