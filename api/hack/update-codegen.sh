#!/bin/bash -e

set -o errexit
set -o nounset
set -o pipefail

echo Removing old clients
rm -rf "pkg/crd/client"

echo "" > /tmp/headerfile

GOPATH=$(go env GOPATH) ./vendor/k8s.io/code-generator/generate-groups.sh all \
    github.com/kubermatic/kubermatic/api/pkg/crd/client github.com/kubermatic/kubermatic/api/pkg/crd \
    "kubermatic:v1" \
    --go-header-file /tmp/headerfile

client-gen --clientset-name clientset \
  --input-base sigs.k8s.io/cluster-api/pkg/apis \
  --input cluster/v1alpha1 \
  --go-header-file /tmp/headerfile \
  --output-package="github.com/kubermatic/kubermatic/api/pkg/client/cluster-api"

rm /tmp/headerfile
