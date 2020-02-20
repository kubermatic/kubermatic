#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
DEFAULT_IMAGE_TAG="$(git rev-parse 'origin/master')"
if [[ -z "${KUBERMATICCOMMIT:-}" && \
    "$(git rev-parse HEAD)" != "${DEFAULT_IMAGE_TAG}" ]]; then
    cat << EOF >&2
WARNING: The KUBERMATICCOMMIT is defaulting to origin/master head commit
and your current HEAD is not on it.

If you are on an old commit and you want to use the images corresponding to
such version run the following command instead:

$ KUBERMATICCOMMIT="$(git rev-parse HEAD)" $0
EOF
fi
# KUBERMATICCOMMIT is used to define the tag of the k8c images used by the
# seed controller.
export KUBERMATICCOMMIT="${KUBERMATICCOMMIT:-${DEFAULT_IMAGE_TAG}}"
make seed-controller-manager

KUBERMATIC_WORKERNAME=${KUBERMATIC_WORKERNAME:-$(uname -n)}
KUBERMATIC_DEBUG=${KUBERMATIC_DEBUG:-true}
PPROF_PORT=${PPROF_PORT:-6600}

./_build/seed-controller-manager \
  -dynamic-datacenters=true \
  -namespace=kubermatic \
  -datacenter-name=europe-west3-c \
  -kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -versions=../config/kubermatic/static/master/versions.yaml \
  -updates=../config/kubermatic/static/master/updates.yaml \
  -kubernetes-addons-path=../addons \
  -kubernetes-addons-file=../config/kubermatic/static/master/kubernetes-addons.yaml \
  -openshift-addons-path=../openshift_addons \
  -openshift-addons-file=../config/kubermatic/static/master/openshift-addons.yaml \
  -feature-gates=OpenIDAuthPlugin=true \
  -worker-name="$(tr -cd '[:alnum:]' <<< $KUBERMATIC_WORKERNAME | tr '[:upper:]' '[:lower:]')" \
  -external-url=dev.kubermatic.io \
  -backup-container=../config/kubermatic/static/store-container.yaml \
  -cleanup-container=../config/kubermatic/static/cleanup-container.yaml \
  -docker-pull-config-json-file=../../secrets/seed-clusters/dev.kubermatic.io/.dockerconfigjson \
  -oidc-ca-file=../../secrets/seed-clusters/dev.kubermatic.io/caBundle.pem \
  -oidc-issuer-url=$(vault kv get -field=oidc-issuer-url dev/seed-clusters/dev.kubermatic.io) \
  -oidc-issuer-client-id=$(vault kv get -field=oidc-issuer-client-id dev/seed-clusters/dev.kubermatic.io) \
  -oidc-issuer-client-secret=$(vault kv get -field=oidc-issuer-client-secret dev/seed-clusters/dev.kubermatic.io) \
  -monitoring-scrape-annotation-prefix='kubermatic.io' \
  -log-debug=$KUBERMATIC_DEBUG \
  -log-format=Console \
  -max-parallel-reconcile=10 \
  -pprof-listen-address=":${PPROF_PORT}" \
  -logtostderr \
  -v=4 # Log-level for the Kube dependencies. Increase up to 9 to get request-level logs.
