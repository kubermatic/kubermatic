#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail


cd ${GOPATH}/src/github.com/kubermatic/api
./_build/kubermatic-api \
  -master-kubeconfig=../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -datacenters=../secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  -kubeconfig=../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -worker-name="$(uname -n | tr -cd '[:alnum:]')" \
  -token-issuer=http://auth.int.kubermatic.io \
  -address=127.0.0.1:8080 \
  -client-id=kubermatic \
  -logtostderr \
  -v=8
