#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

make -C $(dirname $0)/.. master-controller-manager

KUBERMATIC_WORKERNAME=${KUBERMATIC_WORKERNAME:-$(uname -n)}

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
./_build/master-controller-manager \
  -datacenters=../../secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  -kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -internal-address=127.0.0.1:8086 \
  -worker-name="$(tr -cd '[:alnum:]' <<< $KUBERMATIC_WORKERNAME | tr '[:upper:]' '[:lower:]')" \
  -log-debug=true \
  -log-format=Console \
  -logtostderr=1 \
  -v=6 $@
