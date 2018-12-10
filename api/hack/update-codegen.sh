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

# Temporary fixes due to: https://github.com/kubernetes/kubernetes/issues/71655
GENERIC_FILE="$(dirname "$0")/../pkg/crd/client/informers/externalversions/generic.go"
sed -i s/usersshkeys/usersshkeies/g ${GENERIC_FILE}

echo "Generating deepcopy funcs for other packages"
GOPATH=$(go env GOPATH) $(go env GOPATH)/bin/deepcopy-gen \
    --input-dirs github.com/kubermatic/kubermatic/api/pkg/semver \
    -O zz_generated.deepcopy \
    --go-header-file /tmp/headerfile

rm /tmp/headerfile
