#!/usr/bin/env bash

set -euo pipefail
# Required for signal propagation to work so
# the cleanup trap gets executed when the script
# receives a SIGINT
set -o monitor

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api/hack/ci

export UPGRADE_TEST_BASE_HASH=${UPGRADE_TEST_BASE_HASH:-"master"}

# We need to fetch UPGRADE_TEST_BASE_HASH in case its not in either the PRs base or the Prs HEAD
git config --global core.sshCommand 'ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i /ssh/id_rsa'
git remote add origin git@github.com:kubermatic/kubermatic.git
git fetch origin ${UPGRADE_TEST_BASE_HASH}

# Make sure we do not use the local copy of the branches
export UPGRADE_TEST_BASE_HASH=origin/${UPGRADE_TEST_BASE_HASH}

./ci-run-minimal-conformance-tests.sh
