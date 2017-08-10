#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail


cd ${GOPATH}/src/github.com/kubermatic/api
./_build/kubermatic-api \                                                                          
  --worker-name="$(uname -n | tr -cd '[:alnum:]')" \
  --kubeconfig=${GOPATH}/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  --datacenters=${GOPATH}/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  --logtostderr \
  --v=8 \
  --token-issuer=https://kubermatic.eu.auth0.com/ \
  --client-id=xHLUljMUUEFP95wmlODWexe1rvOXuyTT \
  --address=127.0.0.1:8080 \                 
  --master-kubeconfig=${GOPATH}/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig
