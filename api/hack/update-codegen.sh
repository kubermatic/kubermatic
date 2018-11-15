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

echo Generating deepcopy funcs for other packages
GOPATH=$(go env GOPATH) ${GOPATH}/bin/deepcopy-gen \
    --input-dirs github.com/kubermatic/kubermatic/api/pkg/semver \
    -O zz_generated.deepcopy \
    --go-header-file /tmp/headerfile

rm /tmp/headerfile