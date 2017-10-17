#!/bin/bash -e

set -o errexit
set -o nounset
set -o pipefail

echo Removing old clients
rm -rf "pkg/crd/client"

echo "" > /tmp/headerfile

./vendor/k8s.io/code-generator/generate-groups.sh all \
    github.com/kubermatic/kubermatic/api/pkg/crd/client/seed github.com/kubermatic/kubermatic/api/pkg/crd \
    etcdoperator:v1beta2 \
    --go-header-file /tmp/headerfile

./vendor/k8s.io/code-generator/generate-groups.sh all \
    github.com/kubermatic/kubermatic/api/pkg/crd/client/master github.com/kubermatic/kubermatic/api/pkg/crd \
    kubermatic:v1 \
    --go-header-file /tmp/headerfile

rm /tmp/headerfile
