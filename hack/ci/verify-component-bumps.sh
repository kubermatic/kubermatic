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

### KKP releases bump the OSM and machine-controller images together in one PR so
### the conformance suite runs once for both instead of twice (and to avoid a Tide
### base-move retest of the second PR on release branches). This check fails if a PR
### moves only one of the two image Tags, to give fast feedback before the expensive
### e2e suite finishes. It is intentional to bump a single component (the two projects
### release independently); in that case add the "release/single-component-bump" label
### and comment /retest to re-run this check, which then passes.

set -euo pipefail

cd "$(dirname "$0")/../.."
source hack/lib.sh

SKIP_LABEL="release/single-component-bump"

OSM_FILE="pkg/resources/operatingsystemmanager/deployment.go"
MC_FILE="pkg/resources/machinecontroller/deployment.go"

# Prow checks out the PR as a detached merge and does not leave an "origin/$branch"
# ref behind, so diffing against that name silently produces nothing. Use the base
# SHA Prow injects instead, falling back to the merge-base of FETCH_HEAD.
BASE_REF="${PULL_BASE_SHA:-}"
if [ -z "$BASE_REF" ]; then
  BASE_REF="$(git merge-base HEAD FETCH_HEAD 2> /dev/null || true)"
fi
if [ -z "$BASE_REF" ]; then
  echo "Could not determine the base revision (\$PULL_BASE_SHA empty and no FETCH_HEAD)."
  exit 1
fi

echo "Diffing against base ${BASE_REF}"

# did the Tag const change vs the base? compare only the "Tag =" line so an
# unrelated edit elsewhere in the file does not count as a component bump.
tag_line() {
  grep -nE '^[[:space:]]*Tag[[:space:]]*=' "$1" || true
}

tag_changed() {
  local file="$1"
  # diff the file against the base SHA and count changed "Tag =" lines. The +++/---
  # file headers also start with +/- so they are excluded by requiring whitespace or
  # a tab before "Tag". A non-zero count means the Tag const moved.
  # no 2>/dev/null here: a broken diff must fail loudly, not read as "no bump".
  local changed
  changed="$(git diff "${BASE_REF}" HEAD -- "$file" |
    grep -cE '^[+-][[:space:]]+Tag[[:space:]]*=' || true)"
  [ "$changed" -gt 0 ]
}

osm_changed=false
mc_changed=false
tag_changed "$OSM_FILE" && osm_changed=true
tag_changed "$MC_FILE" && mc_changed=true

echo "Component image Tag changes vs ${BASE_REF}:"
printf "  %-18s : %s (%s)\n" "OSM" "$osm_changed" "$(tag_line "$OSM_FILE")"
printf "  %-18s : %s (%s)\n" "machine-controller" "$mc_changed" "$(tag_line "$MC_FILE")"
echo

# both changed, or neither changed: nothing to enforce.
if [ "$osm_changed" = "$mc_changed" ]; then
  echo "OSM and machine-controller Tags are consistent (both or neither bumped)."
  exit 0
fi

# exactly one changed. allow it only if the skip label is present.
# pr_has_label (hack/lib.sh) queries the GitHub API for PULL_NUMBER; same idiom the
# rest of the repo uses, so no extra gh CLI dependency in the CI image.
if pr_has_label "$SKIP_LABEL"; then
  echo "Only one component bumped, but the '$SKIP_LABEL' label is present. Allowing."
  exit 0
fi

cat << EOF
Error: this PR bumps only one of OSM / machine-controller.

KKP releases usually bump both in a single PR so the conformance suite runs once
for both instead of twice (and to avoid a Tide base-move retest of the second PR).

Either:
  - add the other component's image Tag bump to this PR, or
  - if you intentionally need only one, add the label '$SKIP_LABEL'
    and comment /retest to re-run this check.
EOF
exit 1
