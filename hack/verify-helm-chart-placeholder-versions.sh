#!/usr/bin/env bash

# Copyright 2025 The Kubermatic Kubernetes Platform contributors.
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

### This script ensures all Chart.yaml files in the main branch contain the KKP 
### plain semver placeholder 'version: 9.9.9-dev', and are correctly updated 
### during packaging, prior to the GitHub release. Additionally, it exposes
### a list of exceptions, for chart that managed differently within the KKP 
### project (e.g. either vendored from external repositories or maintained 
### separately, and rely on specific packaging instructions)

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

EXPECTED_PLACEHOLDER_VERSION="9.9.9-dev"
CHART_FILES=$(find charts -name "Chart.yaml" | sort)

# Charts that should be excluded from the version check (as they go through custom version patching), but which we expect to find
EXCLUDED_CHARTS=(
  charts/gitops/kkp-argocd-apps/Chart.yaml
  charts/local-kubevirt/Chart.yaml
)

errors=0

echo "Checking Chart.yaml files for version: $EXPECTED_PLACEHOLDER_VERSION"

for chart in $CHART_FILES; do
  # Skip excluded charts
  if printf '%s\n' "${EXCLUDED_CHARTS[@]}" | grep -qx "$chart"; then
    echo "Skipping placeholder version check on Helm chart with custom version management: $chart"
    continue
  fi

  # Check version
  if ! grep -q "^version: ${EXPECTED_PLACEHOLDER_VERSION}$" "$chart"; then
    echo "Error: $chart does not contain expected placeholder Helm chart version '$EXPECTED_PLACEHOLDER_VERSION'"
    errors=1
  fi
done

# Ensure all excluded files exist
for excluded in "${EXCLUDED_CHARTS[@]}"; do
  if [[ ! -f "$excluded" ]]; then
    echo "Error: Expected chart with custom version management was not found: $excluded"
    errors=1
  fi
done

if [[ "$errors" -ne 0 ]]; then
  echo
  echo "Some Chart.yaml files did not meet version expectations."
  exit 1
fi

echo "All Chart.yaml files passed version check."
