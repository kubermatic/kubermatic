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

CONTAINERIZE_IMAGE=quay.io/kubermatic/build:go-1.25-node-22-5 containerize ./hack/update-docs.sh

(
  cd docs
  go run ../codegen/godoc/main.go
)

dummy=kubermaticNoOmitPlease

# temporarily create a vendor folder
go mod vendor

sed="sed"
[ "$(command -v gsed)" ] && sed="gsed"

KKPV1=vendor/k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1

# remove omitempty tags from structs so that genyaml will not skip fields
$sed -i "s/,omitempty/,$dummy/g" $KKPV1/*.go vendor/k8s.io/api/core/v1/*.go

# there are some fields that we do actually want to ignore
$sed -i 's/omitgenyaml/omitempty/g' $KKPV1/*.go

# generate docs for ce version
$sed -i "s/,omitcegenyaml/,omitempty,ce/g" $KKPV1/*.go

# generate docs for CE version
go run -tags ce ./codegen/example-yaml . docs

# Restore original tags
$sed -i "s/omitempty,ce/omitcegenyaml/g" $KKPV1/*.go

# generate docs for EE version
go run -tags ee ./codegen/example-yaml . docs

# revert our changes
$sed -i 's/omitempty/omitgenyaml/g' $KKPV1/*.go
$sed -i "s/,$dummy/,omitempty/g" $KKPV1/*.go vendor/k8s.io/api/core/v1/*.go
