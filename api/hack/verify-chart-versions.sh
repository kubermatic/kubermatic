#!/usr/bin/env bash

# This script is used as a presubmit to check that Helm chart versions
# have been updated if charts have been modified. Without the Prow env
# vars, this script won't run properly.

set -euo pipefail

cd $(dirname $0)/../..

echo "Verifying Helm chart versions..."

charts=$(find config/ -name Chart.yaml | sort)
exitCode=0

[ -n "$charts" ] && while read -r chartYAML; do
  name="$(dirname $(echo "$chartYAML" | cut -d/ -f2-))"
  echo "Verifying $name chart..."

  # if this chart was touched in this PR
  if ! git diff --exit-code --no-patch "$PULL_BASE_SHA...$PULL_PULL_SHA" "$(dirname "$chartYAML")"; then
    oldVersion=""

    # as we scan for charts by looking through the current files, we can always
    # determine the new version (unless the Chart.yaml is missing entirely)
    newVersion="$(yq read "$chartYAML" version)"

    # if the chart already existed, retrieve its old version
    if git show "$PULL_BASE_SHA:$chartYAML" > /dev/null 2>&1; then
      oldVersion=$(git show "$PULL_BASE_SHA:$chartYAML" | yq read - version)
    fi

    if [ "$oldVersion" = "$newVersion" ]; then
      echo "Chart $name was modified but its version ($oldVersion) was not changed. Please adjust $chartYAML."
      exitCode=1
    fi
  fi
done <<< "$charts"

exit $exitCode
