#!/usr/bin/env bash

# Copyright 2024 The Kubermatic Kubernetes Platform contributors.
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

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

# thank you https://stackoverflow.com/a/64390598
function change_minor() {
  local delimiter=.
  local array=($(echo "$1" | tr $delimiter '\n'))
  array[1]=$((array[1] + $2))
  echo $(
    local IFS=$delimiter
    echo "${array[*]}"
  )
}

# determine current and previous versions/branches
CURRENT_BRANCH="${PULL_BASE_REF:-main}"
CURRENT_RELEASE=
PREVIOUS_BRANCH=
PREVIOUS_RELEASE=

# fetch branches
git remote add origin https://github.com/kubermatic/kubermatic
git fetch origin --quiet

if [[ "$CURRENT_BRANCH" == "main" ]]; then
  # find the most recent release branch and return its version, like "v2.25"
  PREVIOUS_RELEASE="$(git branch --remotes --list 'origin/release/v*' --format '%(refname:lstrip=4)' | sort -rV | head -n1)"
  CURRENT_RELEASE="$(change_minor "$PREVIOUS_RELEASE" 1)"
elif [[ "$CURRENT_BRANCH" =~ ^release/v ]]; then
  CURRENT_RELEASE="$(basename "$CURRENT_BRANCH")"
  PREVIOUS_RELEASE="$(change_minor "$CURRENT_RELEASE" -1)"
else
  echodate "Error: PULL_BASE_REF ($PULL_BASE_REF) does not point to main or a release branch."
  exit 1
fi

PREVIOUS_BRANCH="release/$PREVIOUS_RELEASE"

echodate "Current KKP release : $CURRENT_RELEASE ($CURRENT_BRANCH)"
echodate "Previous KKP release: $PREVIOUS_RELEASE ($PREVIOUS_BRANCH)"

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

TEST_NAME="Download envtest binaries"
echodate "Downloading envtest binaries..."

TMP_DIR="$(mktemp -d)"
PATH="$PATH:$TMP_DIR"

download_envtest "$TMP_DIR" "1.33.0"

# restore addons at the previous release state
currentAddons="$(mktemp -d)"
cp -ar addons/* "$currentAddons"

previousAddons="$(mktemp -d)"
rm -rf addons/
git checkout "origin/$PREVIOUS_BRANCH" -- addons/
cp -ar addons/* "$previousAddons"

# check if addons are compatible
echodate "Testing addon upgrade compatibility…"

go_test addon_integration -timeout 30m -tags addon_integration ./pkg/test/addon \
  -previous "$previousAddons" \
  -current "$currentAddons"

echodate "Tests completed successfully!"
