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

### This that when we're in a PR targeting a release branch, the minimum
### Go version used to build KKP is not bumped to a different minor version.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

TARGET_BRANCH="${PULL_BASE_REF:-}"

if [ -z "$TARGET_BRANCH" ]; then
  echo "This check can only work in CI when \$PULL_BASE_REF is set."
  exit 1
fi

if ! [[ "$TARGET_BRANCH" =~ ^release/ ]]; then
  echo "This PR is not targeting a release branch, skipping Go version check."
  exit 0
fi

go_minor_version() {
  grep -E '^go 1' "$1" | sed -E 's/^go (1\.[0-9]+).*$/\1/'
}

go_remote_minor_version() {
  local branch="$1"
  local modulePath="$2"

  local url="https://github.com/$REPO_OWNER/$REPO_NAME/raw/refs/heads/$branch/${modulePath}go.mod"
  wget -O tmp.go.mod --quiet "$url"

  go_minor_version tmp.go.mod
  rm tmp.go.mod
}

newVersions="main module: $(go_minor_version go.mod)"
branchVersions="main module: $(go_remote_minor_version "$TARGET_BRANCH" "")"

if [ -d sdk ]; then
  newVersions="$newVersions, SDK module: $(go_minor_version sdk/go.mod)"
  branchVersions="$branchVersions, SDK module: $(go_remote_minor_version "$TARGET_BRANCH" "sdk/")"
fi

echo "Go releases:"
frmt="  %-15s : %s\n"
printf "$frmt" "$TARGET_BRANCH" "$branchVersions"
printf "$frmt" "This PR" "$newVersions"
echo

if [[ "$branchVersions" != "$newVersions" ]]; then
  echo "Error: Go upgrades are not allowed in release branches."
  exit 1
fi

echo "Go modules are valid."
