#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -x

# Please make sure to port-forward prometheus to localhost:9090 if you want to talk to it.

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
./_build/kubermatic-api \
  -kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -datacenters=../../secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  -versions=../config/kubermatic/static/master/versions.yaml \
  -updates=../config/kubermatic/static/master/updates.yaml \
  -master-resources=../config/kubermatic/static/master \
  -worker-name="$(uname -n | tr -cd '[:alnum:]')" \
  -token-issuer=https://dev.kubermatic.io/dex \
  -internal-address=127.0.0.1:18085 \
  -prometheus-url=http://localhost:9090 \
  -address=127.0.0.1:8080 \
  -client-id=kubermatic \
  -logtostderr \
  -v=8
