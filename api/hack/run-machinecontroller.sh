#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

cd $(go env GOPATH)/src/github.com/kubermatic/machine-controller

KUBECONFIG_MACHINE_CONTROLLER=$(mktemp)
kubectl get secret admin-kubeconfig -o go-template='{{ index .data "kubeconfig" }}' \
  | base64 -d > $KUBECONFIG_MACHINE_CONTROLLER


make machine-controller
./machine-controller \
  -kubeconfig=$KUBECONFIG_MACHINE_CONTROLLER \
  -logtostderr \
  -v=4 \
  -cluster-dns=10.10.10.10 \
  -internal-listen-address=0.0.0.0:8085
