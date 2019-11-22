#!/usr/bin/env bash
set -euo pipefail

SDIR=$(dirname $0)

function cleanup() {
	cat ${SDIR}/../../pkg/test/e2e/api/utils/oidc-proxy-client/_build/oidc-proxy-client-errors

	# Kill all descendant processes
	pkill -P $$
}
trap cleanup EXIT

cat <<EOF >preset.yaml
presets:
  items:
    - metadata:
        name: loodse
      spec:
        azure:
          tenantId: ${AZURE_E2E_TESTS_TENANT_ID}
          subscriptionId: ${AZURE_E2E_TESTS_SUBSCRIPTION_ID}
          clientId: ${AZURE_E2E_TESTS_CLIENT_ID}
          clientSecret: ${AZURE_E2E_TESTS_CLIENT_SECRET}
        digitalocean:
          token: ${DO_E2E_TESTS_TOKEN}
        gcp:
          serviceAccount: ${GOOGLE_SERVICE_ACCOUNT}
        openstack:
          username: ${OS_USERNAME}
          password: ${OS_PASSWORD}
          tenant: ${OS_TENANT_NAME}
          domain: ${OS_DOMAIN}
EOF

export ADDITIONAL_HELM_ARGS="--set=kubermatic.presets=$(cat preset.yaml | base64 -w0)"
source "${SDIR}/ci-setup-kubermatic-in-kind.sh"

cat <<EOF >user.yaml
apiVersion: kubermatic.k8s.io/v1
kind: User
metadata:
  name: c41724e256445bf133d6af1168c2d96a7533cd437618fdbe6dc2ef1fee97acd3
spec:
  email: roxy2@loodse.com
  id: 1413636a43ddc27da27e47614faedff24b4ab19c9d9f2b45dd1b89d9_KUBE
  name: roxy2
  admin: true
EOF
retry 2 kubectl apply -f user.yaml

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
	make run >/dev/null 2>./_build/oidc-proxy-client-errors &
)

# Run e2e tests
echo "running the API E2E tests"
more /etc/hosts
go test -tags=create -timeout 20m ${SDIR}/../../pkg/test/e2e/api -v
go test -tags=e2e ${SDIR}/../../pkg/test/e2e/api -v
