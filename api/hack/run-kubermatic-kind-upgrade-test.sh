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

set -euo pipefail
# Required for signal propagation to work so
# the cleanup trap gets executed when the script
# receives a SIGINT
set -o monitor

. $(dirname $0)/lib.sh

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api/hack/ci

export UPGRADE_TEST_BASE_HASH=${UPGRADE_TEST_BASE_HASH:-"master"}

# We need to fetch UPGRADE_TEST_BASE_HASH in case its not in either the PRs base or the Prs HEAD
ensure_github_host_pubkey
git config --global core.sshCommand 'ssh -o CheckHostIP=no -i /ssh/id_rsa'
git remote add origin git@github.com:kubermatic/kubermatic.git
git fetch origin ${UPGRADE_TEST_BASE_HASH}

# We have to make sure UPGRADE_TEST_BASE_HASH is actually a hash and not a branch because its used
# as the image tag later on. Also make sure we use the branch versions from upstream, as the local version
# may be different.
git checkout origin/${UPGRADE_TEST_BASE_HASH}
export UPGRADE_TEST_BASE_HASH="$(git rev-parse HEAD)"
git checkout -

exec ./ci-kind-e2e.sh
