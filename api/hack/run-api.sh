#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -x


cd ${GOPATH}/src/github.com/kubermatic/kubermatic/api
./_build/kubermatic-api \
  -master-kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -datacenters=../../secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  -kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -versions=../config/kubermatic/static/master/versions.yaml \
  -updates=../config/kubermatic/static/master/updates.yaml \
  -worker-name="$(uname -n | tr -cd '[:alnum:]')" \
  -token-issuer=https://dev.kubermatic.io/dex \
  -prometheus-address=127.0.0.1:18085 \
  -address=127.0.0.1:8080 \
  -client-id=kubermatic \
  -logtostderr \
  -v=8
