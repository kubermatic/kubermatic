#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -x

make -C $(dirname $0)/.. kubermatic-api

KUBERMATIC_WORKERNAME=${KUBERMATIC_WORKERNAME:-$(uname -n)}
KUBERMATIC_DEBUG=${KUBERMATIC_DEBUG:-true}
PPROF_PORT=${PPROF_PORT:-6600}

# Please make sure to set -feature-gates=PrometheusEndpoint=true if you want to use that endpoint.

# Please make sure to set -feature-gates=OIDCKubeCfgEndpoint=true if you want to use that endpoint.
# Note that you would have to pass a few additional flags as well.

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
./_build/kubermatic-api \
  -kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -dynamic-datacenters=true \
  -dynamic-presets=true \
  -versions=../config/kubermatic/static/master/versions.yaml \
  -updates=../config/kubermatic/static/master/updates.yaml \
  -master-resources=../config/kubermatic/static/master \
  -worker-name="$(tr -cd '[:alnum:]' <<< $KUBERMATIC_WORKERNAME | tr '[:upper:]' '[:lower:]')" \
  -internal-address=127.0.0.1:18085 \
  -prometheus-url=http://localhost:9090 \
  -address=127.0.0.1:8080 \
  -oidc-url=https://dev.kubermatic.io/dex \
  -oidc-authenticator-client-id=kubermatic \
  -oidc-issuer-client-id="$(vault kv get -field=oidc-issuer-client-id dev/seed-clusters/dev.kubermatic.io)" \
  -oidc-issuer-client-secret="$(vault kv get -field=oidc-issuer-client-secret dev/seed-clusters/dev.kubermatic.io)" \
  -oidc-issuer-redirect-uri="$(vault kv get -field=oidc-issuer-redirect-uri dev/seed-clusters/dev.kubermatic.io)" \
  -oidc-issuer-cookie-hash-key="$(vault kv get -field=oidc-issuer-cookie-hash-key dev/seed-clusters/dev.kubermatic.io)" \
  -service-account-signing-key="$(vault kv get -field=service-account-signing-key dev/seed-clusters/dev.kubermatic.io)" \
  -log-debug=$KUBERMATIC_DEBUG \
  -pprof-listen-address=":${PPROF_PORT}" \
  -log-format=Console \
  -logtostderr \
  -v=4 $@
