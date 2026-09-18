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

echodate "Updating CA bundle..."
curl -Lo charts/kubermatic-operator/static/ca-bundle.pem https://curl.se/ca/cacert.pem

# The chart's golden master fixtures embed the CA bundle, so they have to be
# regenerated whenever it changes. The test script rewrites the .yaml.out files
# in place and then exits non-zero because they differ from what is committed,
# which is exactly what we want here.
echodate "Regenerating the kubermatic-operator chart fixtures..."
./charts/kubermatic-operator/test/test.sh || true

echodate "Done. Updated .yaml.out files."
