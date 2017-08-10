#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail


cd ${GOPATH}/src/github.com/kubermatic/api
./_build/kubermatic-cluster-controller \
  --datacenters=${GOPATH}/src/github.com/kubermatic/config/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  --kubeconfig=${GOPATH}/src/github.com/kubermatic/config/seed-clusters/dev.kubermatic.io/kubeconfig \
  --versions=${GOPATH}/src/github.com/kubermatic/config/kubermatic/static/master/versions.yaml \
  --updates=${GOPATH}/src/github.com/kubermatic/config/kubermatic/static/master/updates.yaml \
  --master-resources=${GOPATH}/src/github.com/kubermatic/config/kubermatic/static/master \
  --worker-name="$(uname -n | tr -cd '[:alnum:]')" \
  --external-url=dev.kubermatic.io \
  --logtostderr=1 \
  --v=6
