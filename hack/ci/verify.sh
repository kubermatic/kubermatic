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

### This script is used as a presubmit to check that Helm chart versions
### have been updated if charts have been modified. Without the Prow env
### vars, this script won't run properly.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

EXIT_CODE=0

try() {
  local title="$1"
  shift

  heading "$title"
  echo -e "$@\n"

  start_time=$(date +%s)

  set +e
  $@
  exitCode=$?
  set -e

  elapsed_time=$(($(date +%s) - $start_time))
  TEST_NAME="$title" write_junit $exitCode "$elapsed_time"

  if [[ $exitCode -eq 0 ]]; then
    echo -e "\n[${elapsed_time}s] SUCCESS :)"
  else
    echo -e "\n[${elapsed_time}s] FAILED."
    EXIT_CODE=1
  fi

  git reset --hard --quiet
  git clean --force

  echo
}

try "Verify code generation" make verify
try "Spellcheck" make spellcheck
try "Verify go.mod" make check-dependencies
try "Verify chart versions" ./hack/ci/verify-chart-versions.sh
try "Verify legacy kubermatic chart" ./hack/verify-kubermatic-chart.sh
try "Verify generated documentation" ./hack/verify-docs.sh
try "Verify license compatibility" ./hack/verify-licenses.sh
try "Verify Grafana dashboards" ./hack/verify-grafana-dashboards.sh
try "Verify Prometheus rules" ./hack/verify-prometheus-rules.sh
try "Verify User Cluster Prometheus rules" ./hack/ci/verify-user-cluster-prometheus-configs.sh

# -l        list files whose formatting differs from shfmt's
# -d        error with a diff when the formatting differs
# -i uint   indent: 0 for tabs (default), >0 for number of spaces
try "shfmt" shfmt -l -sr -i 2 -d hack

exit $EXIT_CODE
