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

containerize ./hack/update-cert-manager-crds.sh

cd charts/cert-manager/

version=$(yq4 '.appVersion' Chart.yaml)
source=https://github.com/cert-manager/cert-manager/releases/download/$version/cert-manager.crds.yaml
# do not use "crds/" or else Helm will try to install the
# CRDs and then never ever touch them again
file=crd/cert-manager.crds.yaml

echo "# This file has been generated by hack/update-cert-manager-crds.sh, do not edit manually." > $file
echo "" >> $file

set -x
curl -sLo - $source >> $file
