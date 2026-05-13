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

### Detects newly added charts and version bumps in hack/mirror-application-charts.sh
### and mirrors them to the OCI registry.
###
### Runs as a Prow post-submit job when mirror-application-charts.sh changes.
### Detects two types of changes:
###   1. New chart entries added to CHART_URLS / CHART_VERSIONS
###   2. Version bumps in CHART_VERSIONS for existing charts
### Then calls mirror-application-charts.sh for each affected chart.
###
### Detection works by sourcing the current and previous versions of
### mirror-application-charts.sh in a subshell. This relies only on the file
### being valid bash with the two associative arrays - no formatting assumptions.
###
### Environment variables:
### * `DRY_RUN=true` - skip actual mirroring, only print charts that would be mirrored
### * `MAX_HISTORY` - how many commits back to look for a valid previous version of the script (default: 5)

set -euo pipefail

cd "$(dirname "$0")/../.."
source hack/lib.sh

SCRIPT_PATH="hack/mirror-application-charts.sh"
: "${DRY_RUN:=false}"
: "${MAX_HISTORY:=5}" # how many commits back to look for a valid previous version of the script

# temp file holding a previous commit's version of the mirror script. declared
# at file scope (not inside main) so the EXIT trap can resolve $OLD_SCRIPT at
# trigger time -- if this were `local` inside main, the variable would be out
# of scope by the time the trap fires and the temp file would leak.
OLD_SCRIPT=$(mktemp)
trap 'rm -f "$OLD_SCRIPT"' EXIT

# sources the given script file in a subshell and prints sorted "name=version"
# lines from its CHART_VERSIONS array. the subshell isolates the source from
# our environment so set -u, traps, and any other side effects don't leak.
# returns empty output (exit 0) if the file is missing, unreadable, or doesn't
# define CHART_VERSIONS (e.g. truncated, totally unrelated file).
read_chart_versions_from() {
  local script_file="$1"
  [[ -f "$script_file" ]] || return 0

  (
    # shellcheck disable=SC1090
    source "$script_file" > /dev/null 2>&1 || exit 0
    # guard against the sourced file not defining CHART_VERSIONS. without
    # this, set -u would cause "${!CHART_VERSIONS[@]}" to abort the subshell
    # and we'd silently lose pairs we should have read.
    [[ "$(declare -p CHART_VERSIONS 2> /dev/null)" == "declare -A"* ]] || exit 0
    for key in "${!CHART_VERSIONS[@]}"; do
      printf '%s=%s\n' "$key" "${CHART_VERSIONS[$key]}"
    done
  ) | sort
}

# --- Main ----------------------------------------------------------------------
main() {
  echodate "Detecting changes in ${SCRIPT_PATH}..."

  # Get a previous version of the script from git history for comparison and write it to the temp file.
  # Remove the main function invocation from the old script to prevent side effects when sourcing it subsequently.
  # ($SCRIPT_PATH is idempotent, not overwriting already synced chart versions subsequently.)
  if ! git show "HEAD~${MAX_HISTORY}:${SCRIPT_PATH}" 2> /dev/null | sed '/^main "\$@"$/d' > "$OLD_SCRIPT"; then
    echo "WARNING: No previous version of ${SCRIPT_PATH} found in the last ${MAX_HISTORY} commits. Treating all charts as new." >&2
  fi

  local old_pairs new_pairs
  old_pairs=$(read_chart_versions_from "$OLD_SCRIPT")
  new_pairs=$(read_chart_versions_from "$SCRIPT_PATH")

  # any "name=version" line in new but not old is either a newly added chart
  # OR a version bump on an existing chart. both should trigger a mirror.
  local changed_pairs
  changed_pairs=$(comm -13 <(echo "$old_pairs") <(echo "$new_pairs"))

  if [[ -z "$changed_pairs" ]]; then
    echodate "No new charts or version changes detected."
    return 0
  fi

  # extract just the chart names for the loop. the inner script reads the
  # version from CHART_VERSIONS itself when no version arg is passed, so we
  # don't need to forward the version explicitly.
  local charts_to_mirror
  charts_to_mirror=$(echo "$changed_pairs" | cut -d= -f1 | sort -u)

  echodate "Charts to mirror:"
  echo "$changed_pairs" | sed 's/^/  - /'

  local chart version
  while IFS= read -r chart; do
    version=$(echo "$new_pairs" | grep "^${chart}=" | cut -d= -f2)
    if [[ "$DRY_RUN" == "true" ]]; then
      echodate "[DRY-RUN] Would mirror: ${chart} @ ${version}"
    else
      echodate "Mirroring: ${chart} @ ${version}"
      ./"${SCRIPT_PATH}" "${chart}"
    fi
  done <<< "$charts_to_mirror"

  echodate "Done."
}

main "$@"
