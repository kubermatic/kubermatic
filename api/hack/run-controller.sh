#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail


cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
./_build/kubermatic-controller-manager \
  -datacenters=../../secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  -datacenter-name=europe-west3-c \
  -kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -versions=../config/kubermatic/static/master/versions.yaml \
  -updates=../config/kubermatic/static/master/updates.yaml \
  -master-resources=../config/kubermatic/static/master \
  -worker-name="$(uname -n | tr -cd '[:alnum:]')" \
  -external-url=dev.kubermatic.io \
  -logtostderr=1 \
  -v=6
