#!/usr/bin/env bash

set -euo pipefail

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic
source ./api/hack/lib.sh

if [ -z "${KUBERMATIC_DEX_DEV_E2E_USERNAME:-}" ]; then
  echo "KUBERMATIC_DEX_DEV_E2E_USERNAME must be defined."
  exit 1
fi

if [ -z "${KUBERMATIC_DEX_DEV_E2E_PASSWORD:-}" ]; then
  echo "KUBERMATIC_DEX_DEV_E2E_PASSWORD must be defined."
  exit 1
fi

export KUBERMATIC_DEX_VALUES_FILE=$(realpath api/hack/ci/testdata/oauth_values_cloud.yaml)
export KUBERMATIC_OIDC_LOGIN="$KUBERMATIC_DEX_DEV_E2E_USERNAME"
export KUBERMATIC_OIDC_PASSWORD="$KUBERMATIC_DEX_DEV_E2E_PASSWORD"
export KUBERMATIC_SCHEME="https"
export KUBERMATIC_HOST="cloud.kubermatic.io"

echodate "Running Kubermatic API end-to-end tests..."
go test -v -tags=cloud -timeout 25m ./api/pkg/test/e2e/api
