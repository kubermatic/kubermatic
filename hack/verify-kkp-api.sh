#!/usr/bin/env bash

# Copyright 2023 The Kubermatic Kubernetes Platform contributors.
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

# This will re-download the CRDs for the currently used k8c.io/api version.
NO_UPDATE=1 ./hack/update-kkp-api.sh

echodate "Diffing..."
if ! git diff --exit-code pkg; then
  echodate "The k8c.io CRDs are out of date. Please run hack/update-kkp-api.sh."
  exit 1
fi

echodate "CRDs are in-sync."
