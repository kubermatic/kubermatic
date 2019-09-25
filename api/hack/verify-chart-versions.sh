#!/usr/bin/env bash

# This script is used as a presubmit to check that Helm chart versions
# have been updated if charts have been modified. Without the Prow env
# vars, this script won't run properly.

set -euo pipefail

cd $(dirname $0)/../..

charts=$(find config/ -name Chart.yaml | cut -d/ -f2- | sort)
exitCode=0

[ -n "$charts" ] && while read -r chartYAML; do
  chartdir="$(dirname "$chartYAML")"

  # if this chart was touched in this PR
  if ! git diff --exit-code --no-patch "master...$PULL_PULL_SHA" "config/$chartdir/"; then
    oldVersion=""

    # as we scan for charts by looking through the current files, we can always
    # determine the new version (unless the Chart.yaml is missing entirely)
    newVersion="$(yq read "$chartYAML" version)"

    # if the chart already existed, retrieve its old version
    if git show "$PULL_BASE_SHA:$chartYAML" > /dev/null 2>&1; then
      oldVersion=$(git show "$PULL_BASE_SHA:$chartYAML" | yq read - version)
    fi

    if [ "$oldVersion" = "$newVersion" ]; then
      echo "Chart $chartdir was modified but its version ($oldVersion) was not changed. Please adjust $chartYAML."
      exitCode=1
    fi
  fi
done <<< "$charts"

exit $exitCode
