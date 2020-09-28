#!/usr/bin/env bash

# Copyright 2020 The Kubermatic Kubernetes Platform contributors.
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

# This script is run for every tagged revision and will create
# the appropriate GitHub release and upload source archives.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

export DRY_RUN=true
export GITHUB_TOKEN="none"
export GIT_TAG="v2.99.99"
export RELEASE_PLATFORMS="linux-amd64 windows-amd64"

echodate "Simulating GitHub release..."

./hack/ci/github-release.sh

echodate "Simulation successful."

ls -lah _dist
echo

for f in _dist/*; do
  echodate "$(basename "$f")"

  if [[ $f =~ '.zip' ]]; then
    unzip -ql "$f"
  else
    tar tvzf "$f"
  fi

  echo
done
