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

### Checks that every Prometheus metric used by the seed Prometheus rules and
### the Grafana dashboards is listed in the MLA metric contract, and that the
### contract does not list metrics nothing uses any more. This catches
### exporter bumps that rename or drop metrics.

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

ROOT="$(pwd)"
CONTRACT="$ROOT/pkg/test/e2e/testkit/mla/metric-contract.yaml"
TOOL_DIR="$ROOT/hack/tools/promql-metrics"

if [ ! -f "$CONTRACT" ]; then
  echodate "Metric contract $CONTRACT does not exist."
  echo "List the metrics that rules and dashboards reference with:"
  echo
  echo "  (cd hack/tools/promql-metrics && go run . -format json)"
  echo
  echo "Then sort each name into one of the top-level lists of the contract:"
  echo
  echo "  seed:          # scraped by the seed Prometheus in the MLA e2e environment"
  echo "  userCluster:   # sent by user cluster MLA agents in the MLA e2e environment"
  echo "  notInCI:       # list of {name, reason}, never exported in CI"
  exit 1
fi

echodate "Checking MLA metric contract..."
(cd "$TOOL_DIR" && go run . -root "$ROOT" -check -contract "$CONTRACT")
