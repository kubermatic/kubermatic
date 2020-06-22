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

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic
source ./api/hack/lib.sh

export KUBERMATIC_NO_WORKER_NAME=true
export GIT_HEAD_HASH="$(git rev-parse HEAD)"

export KUBERMATIC_VERSION="${UPGRADE_TEST_BASE_HASH:-$GIT_HEAD_HASH}"
echodate "Setting up kubermatic in kind on revision ${KUBERMATIC_VERSION}"

beforeKubermaticSetup=$(nowms)
source ./api/hack/ci/ci-setup-kubermatic-in-kind.sh
pushElapsed kind_kubermatic_setup_duration_milliseconds $beforeKubermaticSetup

echodate "Done setting up kubermatic in kind"

# We can safely assume that the base version got tested already
if [[ -n ${UPGRADE_TEST_BASE_HASH:-} ]]; then
  export ONLY_TEST_CREATION=true
fi

echodate "Running conformance tests"
CHARTS_VERSION="$KUBERMATIC_VERSION" ./api/hack/ci/ci-run-conformance-tester.sh

# No upgradetest, just exit
if [[ -z ${UPGRADE_TEST_BASE_HASH:-} ]]; then
  echodate "Success!"
  exit 0
fi

# Kill OIDC proxy if it still runs to avoid "socket already in use" errors
if [[ -n "$(pidof oidc-proxy-client)" ]]; then
  echodate "oidc-proxy-client still running, killing"
  kill "$(pidof oidc-proxy-client)"
  echodate "Done killing oidc-proxy-client"
fi

export ONLY_TEST_CREATION=false
export KUBERMATIC_VERSION="${GIT_HEAD_HASH}"
export KUBERMATIC_USE_EXISTING_CLUSTER=true
echodate "Setting up kubermatic in kind on revision ${KUBERMATIC_VERSION} for upgradetest"
source ./api/hack/ci/ci-setup-kubermatic-in-kind.sh
echodate "Done setting up kubermatic in kind"

echodate "Getting existing project id"
# Redirect stdout only, so errors are still visible in the log
retry 3 kubectl get project -o name > /tmp/project_id
export KUBERMATIC_PROJECT_ID="$(cat /tmp/project_id | cut -d '/' -f2)"
echodate "Using existing project with id \"${KUBERMATIC_PROJECT_ID}\""

echodate "Running conformance tests a second time"
./api/hack/ci/ci-run-conformance-tester.sh
