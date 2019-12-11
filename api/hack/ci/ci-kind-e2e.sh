#!/usr/bin/env bash

set -euo pipefail

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic
source ./api/hack/lib.sh

export KUBERMATIC_NO_WORKER_NAME=true
export GIT_HEAD_HASH="$(git rev-parse HEAD)"

export KUBERMATIC_VERSION="${UPGRADE_TEST_BASE_HASH:-$(git rev-parse HEAD)}"
echodate "Setting up kubermatic in kind on revision ${KUBERMATIC_VERSION}"
source ./api/hack/ci/ci-setup-kubermatic-in-kind.sh
echodate "Done setting up kubermatic in kind"

echodate "Running conformance tests"
./api/hack/ci/ci-run-conformance-tester.sh

# No upgradetest, just exit
if [[ -z ${UPGRADE_TEST_BASE_HASH:-} ]]; then
  echodate "Success!"
  exit 0
fi

export KUBERMATIC_VERSION="${GIT_HEAD_HASH}"
echodate "Setting up kubermatic in kind on revision ${KUBERMATIC_VERSION} for upgradetest"
source ./api/hack/ci/ci-setup-kubermatic-in-kind.sh
echodate "Done setting up kubermatic in kind"
