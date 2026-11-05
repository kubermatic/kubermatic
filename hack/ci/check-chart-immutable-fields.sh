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

### Compares the golden Helm renders (charts/**/*.yaml.out) of the PR base
### with the working tree and fails when a change touches a field that
### `helm upgrade` cannot change in place, or removes a StatefulSet or
### PersistentVolumeClaim. The base is PULL_BASE_SHA, or origin/main locally.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

base="${PULL_BASE_SHA:-origin/main}"

# prints one "<object>\t<field>\t<value>" line per immutable field; "removed"
# marks objects whose deletion loses data, Helm hooks are recreated anyway
query='
select(.kind != null and .metadata.annotations."helm.sh/hook" == null) |
(.kind + "/" + (.metadata.namespace // "") + "/" + .metadata.name) as $id |
(
  (select(.kind == "StatefulSet") | [
    ["removed", "present"],
    ["spec.selector", (.spec.selector | to_json(0))],
    ["spec.serviceName", (.spec.serviceName | to_json(0))],
    ["spec.podManagementPolicy", (.spec.podManagementPolicy | to_json(0))],
    ["spec.volumeClaimTemplates", ([.spec.volumeClaimTemplates[]? | {"name": .metadata.name, "storageClassName": .spec.storageClassName, "accessModes": .spec.accessModes, "storage": .spec.resources.requests.storage}] | sort_by(.name) | to_json(0))]
  ]),
  (select(.kind == "Deployment" or .kind == "DaemonSet") | [["spec.selector", (.spec.selector | to_json(0))]]),
  (select(.kind == "Service" and (.spec.clusterIP // "") != "") | [["spec.clusterIP", .spec.clusterIP]]),
  (select(.kind == "PersistentVolumeClaim") | [
    ["removed", "present"],
    ["spec.storageClassName", (.spec.storageClassName | to_json(0))],
    ["spec.accessModes", (.spec.accessModes | to_json(0))]
  ]),
  (select(.kind == "Job") | [["spec.template", (.spec.template | to_json(0))]])
) | .[] | $id + "\t" + .[0] + "\t" + .[1]
'

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

failed=0
for file in $(git diff --name-only "$base" -- 'charts/*.yaml.out'); do
  # new golden files have nothing to upgrade from
  git cat-file -e "$base:$file" 2> /dev/null || continue

  git show "$base:$file" | yq eval --no-doc "$query" - > "$tmp/old"
  if [ -f "$file" ]; then
    yq eval --no-doc "$query" "$file" > "$tmp/new"
  else
    : > "$tmp/new"
  fi

  # an object missing from the new render only matters when it holds data;
  # a clusterIP dropped from the chart keeps the allocated one
  if ! awk -F'\t' -v file="$file" '
    FILENAME == ARGV[1] { value[$1 FS $2] = $3; present[$1] = 1; next }
    !($1 in present) {
      if ($2 == "removed") { printf "%s: %s was removed\n", file, $1; bad = 1 }
      next
    }
    !(($1 FS $2) in value) && $2 == "spec.clusterIP" { next }
    value[$1 FS $2] != $3 {
      printf "%s: %s %s changed\n  old: %s\n  new: %s\n", file, $1, $2, $3, value[$1 FS $2]
      bad = 1
    }
    END { exit bad }
  ' "$tmp/new" "$tmp/old"; then
    failed=1
  fi
done

if [ $failed -ne 0 ]; then
  echodate "These changes cannot be applied with helm upgrade. Existing installations need a migration, for example deleting the old object in the installer before the upgrade."
  exit 1
fi

echodate "No immutable chart fields changed."
