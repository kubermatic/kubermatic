#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

KUBERMATIC_WORKERNAME=${KUBERMATIC_WORKERNAME:-$(uname -n)}

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
./_build/rbac-generator \
  -kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -worker-name="$(tr -cd '[:alnum:]' <<< $KUBERMATIC_WORKERNAME | tr '[:upper:]' '[:lower:]')" \
  -logtostderr=1 \
  -v=6 $@
