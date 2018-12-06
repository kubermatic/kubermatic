#!/usr/bin/env bash

set -euo pipefail

export BUILD_ID=${BUILD_ID:-BUILD_ID_UNDEF}

cd $(dirname $0)/..

cd $(go env GOPATH)/src/github.com/kubermatic/secrets
echo $KUBERMATIC_SECRETS_GPG_KEY_BASE64 | base64 -d > /tmp/git-crypt-key
git-crypt unlock /tmp/git-crypt-key
cd -


echo "Building conformance-tests cli and kubermatic-controller-manager"
go build github.com/kubermatic/kubermatic/api/cmd/conformance-tests
make kubermatic-controller-manager

echo "Starting conformance tests"
./conformance-tests \
  -debug \
  -worker-name=$BUILD_ID \
  -kubeconfig=$(go env GOPATH)/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -datacenters=$(go env GOPATH)/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  -kubermatic-nodes=3 \
  -kubermatic-parallel-clusters=11 \
  -kubermatic-delete-cluster=true \
  -name-prefix=prow-e2e \
  -reports-root=/reports \
  -cleanup-on-start=false \
  -aws-access-key-id="$AWS_E2E_TESTS_KEY_ID" \
  -aws-secret-access-key="$AWS_E2E_TESTS_SECRET" \
  -providers=aws \
  -exclude-kubernetes-versions="9,10,11" \
  -exclude-distributions="ubuntu,centos"
