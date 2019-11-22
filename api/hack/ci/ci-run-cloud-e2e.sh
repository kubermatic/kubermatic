#!/usr/bin/env bash

set -euo pipefail

SDIR=$(dirname $0)

export KUBERMATIC_OIDC_CLIENT_ID="kubermatic"
export KUBERMATIC_OIDC_CLIENT_SECRET="ZXhhbXBsZS1hcHAtc2VjcmV0"
export KUBERMATIC_OIDC_ISSUER="https://cloud.kubermatic.io/dex"
export KUBERMATIC_OIDC_REDIRECT_URI="http://localhost:8000"
export KUBERMATIC_OIDC_ISSUER_URL_PREFIX="dex"
export KUBERMATIC_SCHEME="https"
export KUBERMATIC_HOST="cloud.kubermatic.io"

(
	cd ${SDIR}/../../pkg/test/e2e/api/utils/oidc-proxy-client
	make build
)

export KUBERMATIC_OIDC_USER=${KUBERMATIC_DEX_DEV_E2E_USERNAME:-"roxy@loodse.com"}
export KUBERMATIC_OIDC_PASSWORD=${KUBERMATIC_DEX_DEV_E2E_PASSWORD:-"password"}

echo "running the API E2E tests"
go test -v -tags=cloud -timeout 25m ${SDIR}/../../pkg/test/e2e/api
