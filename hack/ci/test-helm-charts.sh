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

### This script runs whenever one or more Helm charts were updated.
### For every changed chart in a PR, it tries to find a `test/` directory
### inside the chart and then execute it. The purpose is to run some
### static tests for our Helm charts, as not all of them are tested
### in the conformance tests.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

# find all changed charts (the pre-kubermatic-verify-charts job ensures
# that all updated charts also change their Chart.yaml)
changedCharts=$(git diff --name-status "${PULL_BASE_SHA}..${PULL_PULL_SHA:-}" 'charts/**/Chart.yaml' | awk '{ print $2 }' | xargs dirname | sort -u)

for chartDirectory in $changedCharts; do
  chartName="$(basename "$chartDirectory")"
  if [ -x "$chartDirectory/test/test.sh" ]; then
    echodate "Testing $chartName chart..."
    $chartDirectory/test/test.sh
    echodate "$chartName tests completed successfully."
  else
    echodate "$chartName chart was changed, but does not have any tests."
  fi
done

echodate "All changed charts with tests have been tested successfully."
