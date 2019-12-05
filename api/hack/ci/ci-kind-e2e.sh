#!/usr/bin/env bash

set -euo pipefail

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic
source ./api/hack/lib.sh

function cleanup {
  echodate "Sleeping for debugging purposes"
  sleep 1h
  kubectl delete service -l "prow.k8s.io/id=$PROW_JOB_ID"

  # Kill all descendant processes
  pkill -P $$

  # Clean up clusters
  kubectl delete cluster --all --ignore-not-found=true
}
trap cleanup EXIT

export ONLY_TEST_CREATION=true

echodate "Setting up kubermatic in kind"
./api/hack/ci/ci-setup-kubermatic-in-kind.sh
echodate "Done setting up kubermatic in kind"

echodate "Running conformance tests"
export KUBERMATIC_APISERVER_ADDRESS="localhost:8080"
export KUBERMATIC_NO_WORKER_NAME=true
export ONLY_TEST_CREATION=true
./api/hack/ci/ci-run-conformance-tester.sh
