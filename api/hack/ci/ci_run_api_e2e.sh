#!/usr/bin/env bash
set -euo pipefail

SDIR=$(dirname $0)

function cleanup {
  cat ${SDIR}/../../pkg/test/e2e/api/utils/oidc-proxy-client/_build/oidc-proxy-client-errors

	# Kill all descendant processes
	pkill -P $$
}
trap cleanup EXIT

source "${SDIR}/ci-setup-kubermatic-in-kind.sh"

# Create and run OIDC proxy client
# TODO: since OIDC_CLIENT_ID and OIDC_CLIENT_SECRET are defined in the docker image
#       they could be exposed as envs by that image
export KUBERMATIC_OIDC_CLIENT_ID="kubermatic"
export KUBERMATIC_OIDC_CLIENT_SECRET="ZXhhbXBsZS1hcHAtc2VjcmV0"
export KUBERMATIC_OIDC_ISSUER="http://dex.oauth:5556"
export KUBERMATIC_OIDC_REDIRECT_URI="http://localhost:8000"
(
cd ${SDIR}/../../pkg/test/e2e/api/utils/oidc-proxy-client
make build
make run > /dev/null 2> ./_build/oidc-proxy-client-errors &
)

# Run e2e tests
echo "running the API E2E tests"
more /etc/hosts
go test -tags=create -timeout 20m ${SDIR}/../../pkg/test/e2e/api -v
go test -tags=e2e ${SDIR}/../../pkg/test/e2e/api -v
