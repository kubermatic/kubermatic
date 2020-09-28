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

# This script is run for every commit to master and will create
# a dummy release in a dedicated GitHub repo. The purpose is to
# a) test that releasing code actually works, and
# b) to provide test builds for internal purposes.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh
rootdir="$(pwd)"

export GIT_TAG="$(git rev-parse HEAD)"

git config --global user.email "dev@kubermatic.com"
git config --global user.name "Prow CI Robot"
git config --global core.sshCommand 'ssh -o CheckHostIP=no -i /ssh/id_rsa'
ensure_github_host_pubkey

# create a nice looking, sortable, somewhat meaningful release name
commitDate="$(git show --pretty='%ct' -s "$GIT_TAG")"
releaseDate="$(date --date="@$commitDate" +"%Y-%m-%d-%H%M%S" --utc)"

export RELEASE_NAME="$releaseDate-$(git rev-parse --short "$GIT_TAG")"
export GIT_REPO="kubermatic/kubermatic-builds"

# create dummy tag in $GIT_REPO
echodate "Tagging $RELEASE_NAME in $GIT_REPO..."

builds="$(mktemp -d)"
git clone "git@github.com:$GIT_REPO.git" "$builds"
cd "$builds"
git tag -f -m "This is just a test version, DO NOT USE." "$RELEASE_NAME"
git push --force origin "$RELEASE_NAME"

# create the actual release
echodate "Creating canary release..."

cd "$rootdir"
./hack/ci/github-release.sh

echodate "Done, find the new release at https://github.com/$GIT_REPO/releases/tag/$RELEASE_NAME"
