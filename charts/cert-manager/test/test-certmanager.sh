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

cd $(dirname $0)/../../..
source hack/lib.sh

# create kind cluster
export KIND_CLUSTER_NAME=cert-manager-tester
export DISABLE_CLUSTER_EXPOSER=yes
source hack/ci/setup-kind-cluster.sh

# try to install cert-manager
echodate "Installing cert-manager..."
kubectl apply -f charts/cert-manager/crd/

helm upgrade \
  --install \
  --namespace cert-manager \
  --create-namespace \
  --atomic \
  cert-manager charts/cert-manager/

if ! which cmctl; then
  echodate "Downloading cmctl..."
  OS=$(go env GOOS); ARCH=$(go env GOARCH); curl -sLo cmctl "https://github.com/cert-manager/cmctl/releases/download/v2.0.0/cmctl_${OS}_${ARCH}"
  chmod +x cmctl

  function cmctl_cleanup {
    echodate "Cleaning up..."
    rm cmctl
  }
  appendTrap cmctl_cleanup EXIT
fi
echodate "Testing cert-manager..."
./cmctl check api --wait=2m
exitcode=$?

echodate "Deleting kind cluster..."
kind delete cluster --name "$KIND_CLUSTER_NAME" || true

exit $exitcode
