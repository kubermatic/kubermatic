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

echodate "Removing old clients"
rm -rf "pkg/crd/client"

echo "" > /tmp/headerfile

echodate "Generating kubermatic:v1"
GOPATH=$(go env GOPATH) ./vendor/k8s.io/code-generator/generate-groups.sh all \
    github.com/kubermatic/kubermatic/api/pkg/crd/client \
    github.com/kubermatic/kubermatic/api/pkg/crd \
    "kubermatic:v1" \
    --go-header-file /tmp/headerfile

echodate "Generating operator:v1alpha1"
GOPATH=$(go env GOPATH) ./vendor/k8s.io/code-generator/generate-groups.sh deepcopy,lister,informer \
    github.com/kubermatic/kubermatic/api/pkg/crd/client \
    github.com/kubermatic/kubermatic/api/pkg/crd \
    "operator:v1alpha1" \
    --go-header-file /tmp/headerfile

# Temporary fixes due to: https://github.com/kubernetes/kubernetes/issues/71655
GENERIC_FILE="pkg/crd/client/informers/externalversions/generic.go"
sed -i s/usersshkeys/usersshkeies/g ${GENERIC_FILE}

echodate "Generating deepcopy funcs for other packages"
GOPATH=$(go env GOPATH) $(go env GOPATH)/bin/deepcopy-gen \
    --input-dirs github.com/kubermatic/kubermatic/api/pkg/semver \
    -O zz_generated.deepcopy \
    --go-header-file /tmp/headerfile

rm /tmp/headerfile

echodate "Generating reconciling functions"
go generate pkg/resources/reconciling/ensure.go
