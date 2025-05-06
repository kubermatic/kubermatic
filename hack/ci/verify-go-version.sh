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

### This that when we're in a PR targetting a release branch, the minimum
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
  echo "This PR is not targetting a release branch, skipping Go version check."
  exit 0
fi

go_minor_version() {
  grep -E '^go 1' "$1" | sed -E 's/^go (1\.[0-9]+).*$/\1/'
}

currentURL="https://github.com/$REPO_OWNER/$REPO_NAME/raw/refs/heads/$TARGET_BRANCH/go.mod"
wget -O current.go.mod --no-verbose "$currentURL"

requested="$(go_minor_version go.mod)"
current="$(go_minor_version current.go.mod)"
rm current.go.mod

echo "$TARGET_BRANCH's Go release: $current"
echo "This PR's Go release......: $requested"
echo

if [[ "$requested" != "$current" ]]; then
  echo "Go upgrades are not allowed in release branches."
  exit 1
fi

echo "go.mod is valid."
