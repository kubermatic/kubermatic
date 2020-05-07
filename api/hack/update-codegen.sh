#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

echo Removing old clients
rm -rf "pkg/crd/client"

echo "" >/tmp/headerfile

GENERATOR=$(go list -f '{{.Dir}}' -m k8s.io/code-generator)

bash "$GENERATOR/generate-groups.sh" all \
    github.com/kubermatic/kubermatic/api/pkg/crd/client \
    github.com/kubermatic/kubermatic/api/pkg/crd \
    "kubermatic:v1" \
    --go-header-file /tmp/headerfile

bash "$GENERATOR/generate-groups.sh" deepcopy,lister,informer \
    github.com/kubermatic/kubermatic/api/pkg/crd/client \
    github.com/kubermatic/kubermatic/api/pkg/crd \
    "operator:v1alpha1" \
    --go-header-file /tmp/headerfile

echo "Generating deepcopy funcs for other packages"
bash "$GENERATOR/generate-groups.sh" deepcopy \
    --input-dirs github.com/kubermatic/kubermatic/api/pkg/semver \
    -O zz_generated.deepcopy \
    --go-header-file /tmp/headerfile

rm /tmp/headerfile

go generate pkg/resources/reconciling/ensure.go
