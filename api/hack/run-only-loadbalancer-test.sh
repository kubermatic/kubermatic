#!/usr/bin/env bash

set -euo pipefail

_build/conformance-tests \
  -debug \
  -kubeconfig=$(go env GOPATH)/src/github.com/kubermatic/secrets/seed-clusters/ci.kubermatic.io/kubeconfig \
  -datacenters=$(go env GOPATH)/src/github.com/kubermatic/secrets/seed-clusters/ci.kubermatic.io/datacenters.yaml \
  -kubermatic-nodes=3 \
  -kubermatic-parallel-clusters=11 \
  -kubermatic-delete-cluster=true \
  -name-prefix=alvaro-e2e \
  -reports-root=/tmp/reports \
  -cleanup-on-start=false \
  -run-kubermatic-controller-manager=true \
  -aws-access-key-id="$AWS_E2E_TESTS_KEY_ID" \
  -aws-secret-access-key="$AWS_E2E_TESTS_SECRET" \
  -versions=1.13.2 \
  -providers=aws \
  -exclude-distributions="ubuntu,centos" \
  -kubermatic-delete-cluster=false
