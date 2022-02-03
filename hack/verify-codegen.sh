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

# This will update both generated code and the CRDs for *.k8c.io
./hack/update-codegen.sh

echodate "Diffing..."
if ! git diff --exit-code pkg charts; then
  echodate "The generated code / CRDs are out of date. Please run hack/update-codegen.sh."
  exit 1
fi

echodate "Generated code is in-sync."
