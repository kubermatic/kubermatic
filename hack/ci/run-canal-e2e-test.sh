#!/usr/bin/env bash

# Copyright 2026 The Kubermatic Kubernetes Platform contributors.
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

### This script runs conformance e2e with Canal CNI.

set -euo pipefail

cd $(dirname $0)/../..

export RELEASES_TO_TEST="${RELEASES_TO_TEST:-1.33}"
export DISTRIBUTIONS="${DISTRIBUTIONS:-ubuntu}"

if [[ -n "${SCENARIO_OPTIONS:-}" ]]; then
  export SCENARIO_OPTIONS="${SCENARIO_OPTIONS},canal"
else
  export SCENARIO_OPTIONS="canal"
fi

./hack/ci/run-e2e-tests.sh
