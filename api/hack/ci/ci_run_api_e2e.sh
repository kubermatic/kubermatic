#!/usr/bin/env bash
set -euo pipefail

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic
source ./api/hack/lib.sh

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

export ADDITIONAL_HELM_ARGS="--set=kubermatic.presets=$(cat preset.yaml|base64 -w0)"
source ./api/hack/ci/ci-setup-kubermatic-in-kind.sh

cat <<EOF > user.yaml
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
echodate "Creating roxy2 user..."
retry 2 kubectl apply -f user.yaml

echodate "Running API E2E tests..."
export KUBERMATIC_DEX_VALUES_FILE=$(realpath api/hack/ci/testdata/oauth_values.yaml)
go test -tags=create -timeout 20m ./api/pkg/test/e2e/api -v
go test -tags=e2e ./api/pkg/test/e2e/api -v
