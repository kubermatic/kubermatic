#!/usr/bin/env bash

set -euo pipefail
# Required for signal propagation to work so
# the cleanup trap gets executed when the script
# receives a SIGINT
set -o monitor

. $(dirname $0)/lib.sh

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api/hack/ci

export UPGRADE_TEST_BASE_HASH=${UPGRADE_TEST_BASE_HASH:-"master"}
export EXCLUDE_DISTRIBUTIONS=ubuntu,centos

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
