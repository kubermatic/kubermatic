#!/usr/bin/env bash

set -euo pipefail

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic
source ./api/hack/lib.sh

echodate "Setting up kubermatic in kind"
source ./api/hack/ci/ci-setup-kubermatic-in-kind.sh
echodate "Done setting up kubermatic in kind"

echodate "Running conformance tests"
./api/hack/ci/ci-run-conformance-tester.sh
