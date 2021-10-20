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

CONTAINERIZE_IMAGE=golang:1.17.1 containerize ./hack/update-codegen.sh

echodate "Creating vendor directory"
go mod vendor
chmod +x vendor/k8s.io/code-generator/generate-groups.sh

echodate "Removing old clients"
rm -rf "pkg/crd/client"

echo "" > /tmp/headerfile

# -trimpath would cause the code generation to fail, so undo the
# Makefile's value and also force mod=vendor here
export "GOFLAGS=-mod=vendor"

echodate "Generating kubermatic:v1"
./vendor/k8s.io/code-generator/generate-groups.sh all \
  k8c.io/kubermatic/v2/pkg/crd/client \
  k8c.io/kubermatic/v2/pkg/crd \
  "kubermatic:v1" \
  --go-header-file /tmp/headerfile

echodate "Generating operator:v1alpha1"
./vendor/k8s.io/code-generator/generate-groups.sh deepcopy,lister,informer \
  k8c.io/kubermatic/v2/pkg/crd/client \
  k8c.io/kubermatic/v2/pkg/crd \
  "operator:v1alpha1" \
  --go-header-file /tmp/headerfile

sed="sed"
[ "$(command -v gsed)" ] && sed="gsed"

# Temporary fixes due to: https://github.com/kubernetes/kubernetes/issues/71655
GENERIC_FILE="v2/pkg/crd/client/informers/externalversions/generic.go"
$sed -i s/usersshkeys/usersshkeies/g ${GENERIC_FILE}

echodate "Generating deepcopy funcs for other packages"
go run k8s.io/code-generator/cmd/deepcopy-gen \
  --input-dirs k8c.io/kubermatic/v2/pkg/semver \
  -O zz_generated.deepcopy \
  --go-header-file /tmp/headerfile

# move files into their correct location, generate-groups.sh does not handle
# non-v1 module names very well
cp -r v2/* .
rm -rf v2/

rm /tmp/headerfile

echodate "Generating reconciling functions"
go generate pkg/resources/reconciling/ensure.go

echodate "Generating openAPI v3 CRDs"
go run sigs.k8s.io/controller-tools/cmd/controller-gen \
  crd \
  object:headerFile=./hack/boilerplate/ce/boilerplate.go.txt \
  paths=./pkg/apis/kubermatic/... \
  output:crd:dir=./charts/kubermatic-operator/crd/k8c.io
