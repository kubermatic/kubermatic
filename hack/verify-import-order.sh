#!/usr/bin/env bash

# Copyright 2022 The Kubermatic Kubernetes Platform contributors.
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

if ! [ -x "$(command -v gimps)" ]; then
  echodate "You need to have gimps installed before running this script. Please install it: https://github.com/xrstf/gimps"
  exit 1
fi

echodate "Sorting import statements..."
gimps .

echodate "Diffing..."
if ! git diff --exit-code; then
  echodate "Some import statements are not properly grouped. Please run https://github.com/xrstf/gimps or sort them manually."
  exit 1
fi

echodate "Your Go import statements are in order :-)"
