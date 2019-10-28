#!/usr/bin/env bash

set -exuo pipefail

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
make kubermatic-operator

KUBERMATIC_WORKERNAME=${KUBERMATIC_WORKERNAME:-$(uname -n)}

./_build/kubermatic-operator \
  -kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -worker-name="$(tr -cd '[:alnum:]' <<< $KUBERMATIC_WORKERNAME | tr '[:upper:]' '[:lower:]')" \
  -log-debug=true \
  -log-format=Console \
  -logtostderr \
  -v=4 # Log-level for the Kube dependencies. Increase up to 9 to get request-level logs.
