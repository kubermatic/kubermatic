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

# This script is used as a presubmit to check that Helm chart versions
# have been updated if charts have been modified. Without the Prow env
# vars, this script won't run properly.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

echodate "Verifying Helm chart versions..."

charts=$(find charts/ -name Chart.yaml | sort)
exitCode=0

[ -n "$charts" ] && while read -r chartYAML; do
  name="$(dirname $(echo "$chartYAML" | cut -d/ -f2-))"
  echodate "Verifying $name chart..."

  # if this chart was touched in this PR
  if ! git diff --exit-code --no-patch "$PULL_BASE_SHA..." "$(dirname "$chartYAML")"; then
    oldVersion=""

    # as we scan for charts by looking through the current files, we can always
    # determine the new version (unless the Chart.yaml is missing entirely)
    newVersion="$(yq read "$chartYAML" version)"

    # if the chart already existed, retrieve its old version
    if git show "$PULL_BASE_SHA:$chartYAML" > /dev/null 2>&1; then
      oldVersion=$(git show "$PULL_BASE_SHA:$chartYAML" | yq read - version)
    fi

    if [ "$oldVersion" = "$newVersion" ]; then
      echodate "Chart $name was modified but its version ($oldVersion) was not changed. Please adjust $chartYAML."
      exitCode=1
    fi
  fi
done <<< "$charts"

exit $exitCode
