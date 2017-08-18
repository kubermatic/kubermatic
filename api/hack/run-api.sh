#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail


cd ${GOPATH}/src/github.com/kubermatic/kubermatic/api
./_build/kubermatic-api \
  -master-kubeconfig=$GOPATH/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -datacenters=$GOPATH/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  -kubeconfig=$GOPATH/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -versions=$GOPATH/src/github.com/kubermatic/kubermatic/config/kubermatic/static/master/versions.yaml \
  -updates=$GOPATH/src/github.com/kubermatic/kubermatic/config/kubermatic/static/master/updates.yaml \
  -worker-name="$(uname -n | tr -cd '[:alnum:]')" \
  -token-issuer=http://auth.int.kubermatic.io \
  -address=127.0.0.1:8080 \
  -client-id=kubermatic \
  -logtostderr \
  -v=8
