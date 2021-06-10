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

helm3 upgrade \
  --install \
  --namespace cert-manager \
  --create-namespace \
  --atomic \
  --values charts/cert-manager/test/test-values.yaml \
  cert-manager charts/cert-manager/

# make sure the webhook works, but before that, give the cainjector some
# time to do its magic and make the webhook ready
sleep 5

echodate "Creating test certificate..."
kubectl apply -f charts/cert-manager/test/certificate.yaml

echodate "Deleting kind cluster..."
kind delete cluster --name "$KIND_CLUSTER_NAME"
