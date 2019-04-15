#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -x

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
# Deploy a user-cluster/ipam-controller for which we actuallly
# have a pushed image
export KUBERMATICCOMMIT="${KUBERMATICCOMMIT:-$(git rev-parse origin/master)}"
make kubermatic-controller-manager

KUBERMATIC_WORKERNAME=${KUBERMATIC_WORKERNAME:-$(uname -n)}

./_build/kubermatic-controller-manager \
  -datacenters=../../secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  -datacenter-name=europe-west3-c \
  -kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -versions=../config/kubermatic/static/master/versions.yaml \
  -updates=../config/kubermatic/static/master/updates.yaml \
  -master-resources=../config/kubermatic/static/master \
  -kubernetes-addons-path=../addons \
  -openshift-addons-path=../openshift_addons \
  -worker-name="$(tr -cd '[:alnum:]' <<< $KUBERMATIC_WORKERNAME | tr '[:upper:]' '[:lower:]')" \
  -external-url=dev.kubermatic.io \
  -backup-container=../config/kubermatic/static/backup-container.yaml \
  -cleanup-container=../config/kubermatic/static/cleanup-container.yaml \
  -docker-pull-config-json-file=../../secrets/seed-clusters/dev.kubermatic.io/.dockerconfigjson \
  -oidc-ca-file=../../secrets/seed-clusters/dev.kubermatic.io/caBundle.pem \
  -oidc-issuer-url=$(vault kv get -field=oidc-issuer-url dev/seed-clusters/dev.kubermatic.io) \
  -oidc-issuer-client-id=$(vault kv get -field=oidc-issuer-client-id dev/seed-clusters/dev.kubermatic.io) \
  -oidc-issuer-client-secret=$(vault kv get -field=oidc-issuer-client-secret dev/seed-clusters/dev.kubermatic.io) \
  -monitoring-scrape-annotation-prefix='kubermatic.io' \
  -logtostderr=1 \
  -v=6 $@ 2>&1|tee /tmp/kubermatic-controller-manager.log
