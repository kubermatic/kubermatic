#!/usr/bin/env bash

set -euo pipefail

export BUILD_ID=${BUILD_ID:-BUILD_ID_UNDEF}
echo "Build ID is $BUILD_ID"

cd $(dirname $0)/..

echo "Unlocking secrets repo"
cd $(go env GOPATH)/src/github.com/kubermatic/secrets
echo $KUBERMATIC_SECRETS_GPG_KEY_BASE64 | base64 -d > /tmp/git-crypt-key
git-crypt unlock /tmp/git-crypt-key
cd -
echo "Successfully unlocked secrets repo"


echo "Building conformance-tests cli and kubermatic-controller-manager"
go build github.com/kubermatic/kubermatic/api/cmd/conformance-tests
make kubermatic-controller-manager
echo "Finished building conformance-tests and kubermatic-controller-manager"

function cleanup {
  set +e
  export KUBECONFIG="$(go env GOPATH)/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig"
  # Delete all clusters that have our worker-name label
  kubectl delete cluster -l worker-name=$BUILD_ID --wait=false

  # Remove the worker-name label from all clusters that have our worker-name
  # label so the main cluster-controller will clean them up
  kubectl get cluster -l worker-name=$BUILD_ID \
    -o go-template='{{range .items}}{{.metadata.name}}{{end}}' \
      |xargs -I ^ kubectl label cluster ^ worker-name-
}
trap cleanup EXIT

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
