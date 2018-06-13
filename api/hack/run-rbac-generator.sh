#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail


cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
./_build/rbac-generator \
  -kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -worker-name="$(uname -n | tr -cd '[:alnum:]' | tr '[:upper:]' '[:lower:]')" \
  -logtostderr=1 \
  -v=6 $@
