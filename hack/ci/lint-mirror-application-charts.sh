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

### Lints hack/mirror-application-charts.sh by sourcing it and asserting
### CHART_URLS and CHART_VERSIONS have identical key sets. Also verifies that
### optional CHART_ADDITIONAL_VERSIONS entries reference known charts.
###
### Catches the most common contributor mistake: adding a chart to one
### array but forgetting the other, or adding additional versions for an unknown
### chart. Also serves as a smoke test that mirror-application-charts.sh remains
### sourceable (which the post-submit detection wrapper relies on).

set -euo pipefail

cd "$(dirname "$0")/../.."

SCRIPT="hack/mirror-application-charts.sh"

# strip main "$@" so sourcing only loads arrays/functions without side effects.
# this handles both the old bare call and the new sourcing guard.
stripped=$(mktemp)
trap 'rm -f "$stripped"' EXIT
sed '/^main "\$@"$/d' "$SCRIPT" > "$stripped"

# source in a subshell to keep our env clean, then print sorted key lists
# from all relevant arrays. awk compares the key sets and prints mismatches.
mismatch=$(
  (
    # shellcheck disable=SC1090
    source "$stripped" > /dev/null 2>&1
    {
      printf 'urls\n'
      for k in "${!CHART_URLS[@]}"; do printf '%s\n' "$k"; done | sort
      printf 'versions\n'
      for k in "${!CHART_VERSIONS[@]}"; do printf '%s\n' "$k"; done | sort
      printf 'additional_versions\n'
      if [[ "$(declare -p CHART_ADDITIONAL_VERSIONS 2> /dev/null)" == "declare -A"* ]]; then
        for k in "${!CHART_ADDITIONAL_VERSIONS[@]}"; do printf '%s\n' "$k"; done | sort
      fi
    }
  ) | awk '
    /^urls$/                { mode="urls"; next }
    /^versions$/            { mode="versions"; next }
    /^additional_versions$/ { mode="additional_versions"; next }
    mode=="urls"                { urls[$0]=1 }
    mode=="versions"            { versions[$0]=1 }
    mode=="additional_versions" { additional_versions[$0]=1 }
    END {
      for (k in urls)     if (!(k in versions)) print "missing in CHART_VERSIONS: " k
      for (k in versions) if (!(k in urls))     print "missing in CHART_URLS: "    k
      for (k in additional_versions) if (!(k in urls)) print "unknown chart in CHART_ADDITIONAL_VERSIONS: " k
    }
  '
)

if [[ -n "$mismatch" ]]; then
  echo "FAIL: ${SCRIPT} has inconsistent CHART_URLS / CHART_VERSIONS keys:"
  echo "$mismatch" | sed 's/^/  /'
  echo
  echo "CHART_URLS and CHART_VERSIONS must contain the same chart names; CHART_ADDITIONAL_VERSIONS may only reference those charts."
  exit 1
fi

read -r chart_count additional_count < <(
  (
    # shellcheck disable=SC1090
    source "$stripped" > /dev/null 2>&1
    additional_count=0
    if [[ "$(declare -p CHART_ADDITIONAL_VERSIONS 2> /dev/null)" == "declare -A"* ]]; then
      for k in "${!CHART_ADDITIONAL_VERSIONS[@]}"; do
        for version in ${CHART_ADDITIONAL_VERSIONS[$k]}; do
          [[ -n "$version" ]] && additional_count=$((additional_count + 1))
        done
      done
    fi

    echo "${#CHART_URLS[@]} ${additional_count}"
  )
)
echo "OK: ${SCRIPT}: ${chart_count} charts, ${additional_count} additional chart version(s), chart keys are valid."
