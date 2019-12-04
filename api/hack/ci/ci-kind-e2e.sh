#!/usr/bin/env bash

set -euo pipefail

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic
source ./api/hack/lib.sh

function cleanup {
	kubectl delete service -l "prow.k8s.io/id=$PROW_JOB_ID"

	# Kill all descendant processes
	pkill -P $$

  # Clean up clusters
  kubectl delete cluster --all
}
trap cleanup EXIT

export ONLY_TEST_CREATION=true

echodate "Setting up kubermatic in kind"
./api/hack/ci/ci-setup-kubermatic-in-kind.sh
echodate "Done setting up kubermatic in kind"

echodate "Running conformance tests"
./api/hack/ci/ci-run-conformance-tester.sh
