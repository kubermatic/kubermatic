#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail


cd ${GOPATH}/src/github.com/kubermatic/api
./kubermatic-cluster-controller \
  --datacenters=${GOPATH}/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  --kubeconfig=${GOPATH}/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  --worker-name="$(uname -n | tr -cd '[:alnum:]')" \
  --logtostderr=1 \
  --master-resources=${GOPATH}/src/github.com/kubermatic/config/kubermatic/static/master \
  --v=4 \
  --external-url=dev.kubermatic.io
