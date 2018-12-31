#!/usr/bin/env bash
set -euo pipefail

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api/cmd/e2e
rm -rf tests
mkdir tests

go test \
  -parallel 999 \
  -tags=e2e \
  -run=TestE2E/.*/aws/coreos \
  github.com/kubermatic/kubermatic/api/cmd/e2e \
  -kubeconfig=/home/henrik/go/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -datacenters=/home/henrik/go/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  -test-bin-dir=/opt/kube-test/ \
  -timeout=6h \
  -working-dir=/home/henrik/go/src/github.com/kubermatic/kubermatic/api/cmd/e2e/tests \
  -aws-access-key-id="$(vault kv get -field=accessKeyID dev/e2e-aws)" \
  -aws-secret-access-key="$(vault kv get -field=secretAccessKey dev/e2e-aws)"
