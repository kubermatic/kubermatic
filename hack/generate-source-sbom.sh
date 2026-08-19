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

### Generates a source-tree SBOM covering this repository's Go modules
### and Helm chart dependencies (charts/*/Chart.lock), for internal
### supply-chain visibility. Not a CRA-required artifact.
###
### Usage: hack/generate-source-sbom.sh <release-name> <output-dir>

set -euo pipefail

cd $(dirname "$0")/..
source hack/lib.sh

RELEASE_NAME="${1:?release name is required}"
OUTPUT_DIR="${2:?output directory is required}"

mkdir -p "$OUTPUT_DIR"

outFile="$OUTPUT_DIR/kubermatic-$RELEASE_NAME.sbom.spdx.json"
goSbom="$(mktemp)"
chartDeps="$(mktemp)"

echodate "Generating source SBOM for Go modules..."
syft . \
  --source-name kubermatic \
  --source-version "$RELEASE_NAME" \
  --exclude './_build/**' \
  --exclude './_dist/**' \
  --exclude './_cache/**' \
  --exclude './_artifacts/**' \
  --exclude './.git/**' \
  -o "spdx-json=$goSbom"

echodate "Extracting Helm chart dependencies from charts/*/Chart.lock..."
echo '[]' > "$chartDeps"
for chartLock in charts/*/Chart.lock; do
  [ -f "$chartLock" ] || continue

  chartName="$(basename "$(dirname "$chartLock")")"

  namesRaw="$(yq '.dependencies.[].name' "$chartLock")"
  versionsRaw="$(yq '.dependencies.[].version' "$chartLock")"
  reposRaw="$(yq '.dependencies.[].repository' "$chartLock")"

  depNames=()
  depVersions=()
  depRepos=()
  [ -n "$namesRaw" ] && mapfile -t depNames <<< "$namesRaw"
  [ -n "$versionsRaw" ] && mapfile -t depVersions <<< "$versionsRaw"
  [ -n "$reposRaw" ] && mapfile -t depRepos <<< "$reposRaw"

  chartEntries="[]"
  for idx in "${!depNames[@]}"; do
    repo="${depRepos[$idx]:-}"
    [ "$repo" = "null" ] && repo=""

    entry="$(jq -n \
      --arg chart "$chartName" \
      --arg name "${depNames[$idx]}" \
      --arg version "${depVersions[$idx]}" \
      --arg repo "$repo" \
      '{chart: $chart, name: $name, version: $version, repository: $repo}')"

    chartEntries="$(echo "$chartEntries" | jq --argjson e "$entry" '. + [$e]')"
  done

  jq --argjson entries "$chartEntries" '. + $entries' "$chartDeps" > "$chartDeps.tmp"
  mv "$chartDeps.tmp" "$chartDeps"
done

echodate "Merging Go packages with Helm chart packages and relationships into $outFile..."
jq --slurpfile deps "$chartDeps" '
  ($deps[0] | to_entries) as $indexed
  | (
      $indexed | map({
        "SPDXID": ("SPDXRef-Package-helm-" + (.key | tostring) + "-" + .value.chart + "-" + .value.name),
        "name": .value.name,
        "versionInfo": .value.version,
        "downloadLocation": ((.value.repository // "") | if . == "" then "NOASSERTION" else . end),
        "supplier": "NOASSERTION",
        "externalRefs": []
      })
    ) as $newPackages
  | (
      $indexed | map({
        "spdxElementId": "SPDXRef-DOCUMENT",
        "relationshipType": "DESCRIBES",
        "relatedSpdxElement": ("SPDXRef-Package-helm-" + (.key | tostring) + "-" + .value.chart + "-" + .value.name)
      })
    ) as $newRelationships
  | .packages += $newPackages
  | .relationships += $newRelationships
' "$goSbom" > "$outFile"

rm -f "$goSbom" "$chartDeps"

echodate "Source SBOM written to $outFile"
