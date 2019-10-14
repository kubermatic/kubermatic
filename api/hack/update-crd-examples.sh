#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/..

# remove omitempty tags from structs so that goyaml
# will not skip fields
sed -i 's/,omitempty//g' pkg/crd/kubermatic/v1/*.go
sed -i 's/,omitempty//g' vendor/k8s.io/api/core/v1/*.go

go run codegen/seed-yaml/main.go > ../docs/zz_generated.seed-cr.yaml

git checkout pkg/crd/kubermatic/v1
git checkout vendor/k8s.io/api/core/v1
