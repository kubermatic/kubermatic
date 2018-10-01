#!/usr/bin/env bash

set -euxo pipefail

cd $(dirname $0)/../../../..

TEMPDIR=$(mktemp -d)

type client-gen &>/dev/null || go build -o $(go env GOPATH)/bin/client-gen github.com/kubermatic/kubermatic/api/vendor/k8s.io/code-generator/cmd/client-gen

ls $(go env GOPATH)/src/sigs.k8s.io/cluster-api &>/dev/null || git clone https://github.com/kubernetes-sigs/cluster-api.git $(go env GOPATH)/src/sigs.k8s.io/cluster-api

touch ${TEMPDIR}/header.txt

client-gen --clientset-name clientset \
  --input-base sigs.k8s.io/cluster-api/pkg/apis \
  --input cluster/v1alpha1 \
  --output-base=${TEMPDIR} \
  --go-header-file=${TEMPDIR}/header.txt \
  --output-package="sigs.k8s.io/cluster-api/pkg/client/clientset_generated"

mv ${TEMPDIR}/sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/fake/* \
  vendor/sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/fake/

rm -rf ${TEMPDIR}
