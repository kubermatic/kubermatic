#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/..

# remove omitempty tags from structs so that genyaml will not skip fields
sed -i 's/,omitempty//g' pkg/crd/kubermatic/v1/*.go
sed -i 's/,omitempty//g' vendor/k8s.io/api/core/v1/*.go

# there are some fields that we do actually want to ignore
sed -i 's/omitgenyaml/omitempty/g' pkg/crd/kubermatic/v1/*.go

go run cmd/example-yaml-generator/main.go . ../docs

# revert our changes
git checkout pkg/crd/kubermatic/v1
git checkout vendor/k8s.io/api/core/v1
