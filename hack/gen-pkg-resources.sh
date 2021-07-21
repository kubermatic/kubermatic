#!/usr/bin/env bash

# Copyright 2021 The Kubermatic Kubernetes Platform contributors.
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

cd "$(dirname $0)/.."

tmp_dir=$(mktemp -d)
trap '{ rm -rf -- "$tmp_dir"; }' EXIT

(
    cd "$tmp_dir"
    git clone --branch=v1.21.1 --depth 1 https://github.com/kubernetes/cloud-provider-openstack
    helm template cinder-csi --namespace=kube-system ./cloud-provider-openstack/charts/cinder-csi-plugin > "$tmp_dir/cinder-csi-resources.yaml"
)

go run github.com/wozniakjan/reverse-kube-resource \
    -package=csicinder \
    -go-header-file=./hack/boilerplate/ce/boilerplate.go.txt \
    -src="$tmp_dir/cinder-csi-resources.yaml" > ./pkg/resources/csicinder/resources_gen.go

go run github.com/wozniakjan/reverse-kube-resource \
    -package=csicinder \
    -go-header-file=./hack/boilerplate/ce/boilerplate.go.txt \
    -kubermatic-interfaces \
    -src="$tmp_dir/cinder-csi-resources.yaml" > ./pkg/resources/csicinder/interfaces_gen.go
