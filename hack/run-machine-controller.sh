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

cd $(go env GOPATH)/src/github.com/kubermatic/machine-controller

KUBECONFIG_MACHINE_CONTROLLER=$(mktemp)
kubectl get secret admin-kubeconfig -o go-template='{{ index .data "kubeconfig" }}' |
  base64 -d > $KUBECONFIG_MACHINE_CONTROLLER

make machine-controller
./machine-controller \
  -kubeconfig=$KUBECONFIG_MACHINE_CONTROLLER \
  -logtostderr \
  -v=4 \
  -cluster-dns=10.240.16.10 \
  -internal-listen-address=0.0.0.0:8085
