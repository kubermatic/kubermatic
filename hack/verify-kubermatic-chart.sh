#!/usr/bin/env bash

# This script is used as a presubmit to check that the Kubermatic Helm chart
# matches the default values from the Kubermatic Operator.

set -euo pipefail

cd $(dirname $0)/../..
source api/hack/lib.sh

./api/hack/update-kubermatic-chart.sh

echodate "Verifying Kubermatic Helm chart..."
git diff --exit-code
echodate "No changes detected."
