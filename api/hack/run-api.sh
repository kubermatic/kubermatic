#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -x

make -C $(dirname $0)/.. kubermatic-api

KUBERMATIC_WORKERNAME=${KUBERMATIC_WORKERNAME:-$(uname -n)}

# Please make sure to set -feature-gates=PrometheusEndpoint=true if you want to use that endpoint.

# Please make sure to set -feature-gates=OIDCKubeCfgEndpoint=true if you want to use that endpoint.
# Note that you would have to pass a few additional flags as well.

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
./_build/kubermatic-api \
  -kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -datacenters=../../secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  -versions=../config/kubermatic/static/master/versions.yaml \
  -updates=../config/kubermatic/static/master/updates.yaml \
  -master-resources=../config/kubermatic/static/master \
  -worker-name="$(tr -cd '[:alnum:]' <<< $KUBERMATIC_WORKERNAME | tr '[:upper:]' '[:lower:]')" \
  -internal-address=127.0.0.1:18085 \
  -prometheus-url=http://localhost:9090 \
  -address=127.0.0.1:8080 \
  -oidc-url=https://dev.kubermatic.io/dex \
  -oidc-authenticator-client-id=kubermatic \
  -logtostderr \
  -v=8 $@
