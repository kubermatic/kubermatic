#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/..

go run cmd/addon-godoc-generator/main.go > ../docs/zz_generated.addondata.go

dummy=kubermaticNoOmitPlease

# remove omitempty tags from structs so that genyaml will not skip fields
sed -i "s/,omitempty/,$dummy/g" pkg/crd/kubermatic/v1/*.go pkg/crd/operator/v1alpha1/*.go vendor/k8s.io/api/core/v1/*.go

# there are some fields that we do actually want to ignore
sed -i 's/omitgenyaml/omitempty/g' pkg/crd/kubermatic/v1/*.go pkg/crd/operator/v1alpha1/*.go

go run cmd/example-yaml-generator/main.go . ../docs

# revert our changes
sed -i 's/omitempty/omitgenyaml/g' pkg/crd/kubermatic/v1/*.go pkg/crd/operator/v1alpha1/*.go
sed -i "s/,$dummy/,omitempty/g" pkg/crd/kubermatic/v1/*.go pkg/crd/operator/v1alpha1/*.go vendor/k8s.io/api/core/v1/*.go
