#!/bin/sh

worker="realfake"

set -xe
make 

./_build/kubermatic-api \
  -worker-name="${worker}" \
  -kubeconfig=$GOPATH/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -datacenters=$GOPATH/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  -logtostderr \
  -v=9 \
  -token-issuer=https://kubermatic.eu.auth0.com/ \
  -client-id=xHLUljMUUEFP95wmlODWexe1rvOXuyTT \
  -address=127.0.0.1:8080 \
  -master-kubeconfig=$GOPATH/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig &


./_build/kubermatic-cluster-controller \
  -datacenters=$GOPATH/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  -kubeconfig=$GOPATH/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -logtostderr=1 \
  -master-resources=$GOPATH/src/github.com/kubermatic/config/kubermatic/static/master \
  -v=6 \
  -worker-name="${worker}" \
  -external-url=dev.kubermatic.io \
  -versions=$GOPATH/src/github.com/kubermatic/config/kubermatic/static/master/versions.yaml \
  -updates=$GOPATH/src/github.com/kubermatic/config/kubermatic/static/master/updates.yaml &

wait
