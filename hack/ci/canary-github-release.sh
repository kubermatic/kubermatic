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

### This script is run for every commit to master and will create
### a dummy release in a dedicated GitHub repo. The purpose is to
###
### * test that releasing code actually works, and
### * to provide test builds for internal purposes.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh
rootdir="$(pwd)"

export GIT_TAG="$(git rev-parse HEAD)"

# PULL_BASE_REF is the name of the current branch in case of a post-submit
# or the name of the base branch in case of a PR. Since this is running
# for untagged revisions, we cannot refer to the same revision in the
# dashboard and must instead get the dashboard's latest revision.
export DASHBOARD_GIT_TAG="$(get_latest_dashboard_hash "$PULL_BASE_REF")"

git config --global user.email "dev@kubermatic.com"
git config --global user.name "Prow CI Robot"
git config --global core.sshCommand 'ssh -o CheckHostIP=no -i /ssh/id_rsa'
ensure_github_host_pubkey

# create a nice looking, sortable, somewhat meaningful release name
commitDate="$(git show --pretty='%ct' -s "$GIT_TAG")"
releaseDate="$(date --date="@$commitDate" +"%Y-%m-%d-%H%M%S" --utc)"
shortRef="$(git rev-parse --short "$GIT_TAG")"

export RELEASE_NAME="$releaseDate-$shortRef"
export GIT_REPO="kubermatic/kubermatic-builds"

# prepare a nice looking release description
gitSubject="$(git log --max-count=1 --format="format:%s")"
warning="This is a test release, *do not use in production.*"
export RELEASE_DESCRIPTION="$(echo -e "$warning\n\n* [kubermatic/kubermatic@$shortRef] $gitSubject" | sed --regexp-extended 's;\(#([0-9]+)\)$;(kubermatic/kubermatic#\1);')"

# create dummy tag in $GIT_REPO
echodate "Tagging $RELEASE_NAME in $GIT_REPO..."

builds="$(mktemp --directory)"
git clone "git@github.com:$GIT_REPO.git" "$builds"
cd "$builds"
git tag --force --message "This is just a test version, DO NOT USE." "$RELEASE_NAME"
git push --force origin "$RELEASE_NAME"

# create the actual release
echodate "Creating canary release..."

cd "$rootdir"
./hack/ci/github-release.sh

echodate "Done, find the new release at https://github.com/$GIT_REPO/releases/tag/$RELEASE_NAME"
