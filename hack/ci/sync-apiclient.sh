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

### This script is run as a postsubmit to copy the generated API client
### into the github.com/kubermatic/go-kubermatic repository.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

URL="git@github.com:kubermatic/go-kubermatic.git"

commit_and_push() {
  local repodir branch source_sha
  repodir="$1"
  branch="$2"
  source_sha="$3"

  (
    cd "$repodir"

    git config --local user.email "dev@kubermatic.com"
    git config --local user.name "Prow CI Robot"
    git config --local core.sshCommand 'ssh -o CheckHostIP=no -i /ssh/id_rsa'

    git add .
    if ! git status | grep -q 'nothing to commit'; then
      git commit -m "Syncing client from Kubermatic $source_sha"
      git push origin "$branch"
    fi
  )
}

if [ -z "${PULL_BASE_REF=}" ]; then
  echo "\$PULL_BASE_REF undefined, I don't know which branch to sync."
  exit 1
fi

if [ -n "$(git rev-parse --show-prefix)" ]; then
  echo "You must run this script from repo root"
  exit 1
fi

# Ensure Github's host key is available and disable IP checking.
ensure_github_host_pubkey

# clone the target and pick the right branch
tempdir="$(mktemp -d)"
trap "rm -rf '$tempdir'" EXIT
GIT_SSH_COMMAND="ssh -o CheckHostIP=no -i /ssh/id_rsa" git clone "$URL" "$tempdir"
(
  cd "$tempdir"
  git checkout "$PULL_BASE_REF" || git checkout -b "$PULL_BASE_REF"
)

# rewrite all the import paths
echo "Rewriting import paths"
sed_expression="s#k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient#github.com/kubermatic/go-kubermatic#g"
time find pkg/test/e2e/utils/apiclient/ -type f -exec sed "$sed_expression" -i {} \;

# sync the files
echo "Synchronizing the files"
rsync --archive --verbose --delete "./pkg/test/e2e/utils/apiclient/client/" "$tempdir/client/"
rsync --archive --verbose --delete "./pkg/test/e2e/utils/apiclient/models/" "$tempdir/models/"

# commit and push
commit_and_push "$tempdir" "$PULL_BASE_REF" "$(git rev-parse HEAD)"
