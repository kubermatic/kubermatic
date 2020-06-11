#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

make -C $(dirname $0)/.. master-controller-manager

KUBERMATIC_WORKERNAME=${KUBERMATIC_WORKERNAME:-$(uname -n)}
KUBERMATIC_DEBUG=${KUBERMATIC_DEBUG:-true}
PPROF_PORT=${PPROF_PORT:-6600}

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
./_build/master-controller-manager \
  -dynamic-datacenters=true \
  -kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -internal-address=127.0.0.1:8086 \
  -worker-name="$(tr -cd '[:alnum:]' <<< $KUBERMATIC_WORKERNAME | tr '[:upper:]' '[:lower:]')" \
  -log-debug=$KUBERMATIC_DEBUG \
  -pprof-listen-address=":${PPROF_PORT}" \
  -log-format=Console \
	-logtostderr \
	-v=4 # Log-level for the Kube dependencies. Increase up to 9 to get request-level logs.
