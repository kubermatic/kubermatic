#!/usr/bin/env bash

# Copyright 2020 The Kubermatic Kubernetes Platform contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

CONTAINERIZE_IMAGE=golang:1.17.1 containerize ./hack/update-docs.sh

go run cmd/addon-godoc-generator/main.go > docs/zz_generated.addondata.go.txt

dummy=kubermaticNoOmitPlease

# temporarily create a vendor folder
go mod vendor

# remove omitempty tags from structs so that genyaml will not skip fields
sed -i "s/,omitempty/,$dummy/g" pkg/crd/kubermatic/v1/*.go pkg/crd/operator/v1alpha1/*.go vendor/k8s.io/api/core/v1/*.go

# there are some fields that we do actually want to ignore
sed -i 's/omitgenyaml/omitempty/g' pkg/crd/kubermatic/v1/*.go pkg/crd/operator/v1alpha1/*.go

go run cmd/example-yaml-generator/main.go . docs

# revert our changes
sed -i 's/omitempty/omitgenyaml/g' pkg/crd/kubermatic/v1/*.go pkg/crd/operator/v1alpha1/*.go
sed -i "s/,$dummy/,omitempty/g" pkg/crd/kubermatic/v1/*.go pkg/crd/operator/v1alpha1/*.go vendor/k8s.io/api/core/v1/*.go
