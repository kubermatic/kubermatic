#!/usr/bin/env bash
set -euo pipefail

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic
source ./api/hack/lib.sh

source ./api/hack/ci/ci-setup-kubermatic-in-kind.sh

echodate "Creating UI Azure preset..."
cat <<EOF > preset-azure.yaml
apiVersion: kubermatic.k8s.io/v1
kind: Preset
metadata:
  name: e2e-azure
  namespace: kubermatic
spec:
  azure:
    tenantId: ${AZURE_E2E_TESTS_TENANT_ID}
    subscriptionId: ${AZURE_E2E_TESTS_SUBSCRIPTION_ID}
    clientId: ${AZURE_E2E_TESTS_CLIENT_ID}
    clientSecret: ${AZURE_E2E_TESTS_CLIENT_SECRET}
EOF
retry 2 kubectl apply -f preset-azure.yaml

echodate "Creating UI DigitalOcean preset..."
cat <<EOF > preset-digitalocean.yaml
apiVersion: kubermatic.k8s.io/v1
kind: Preset
metadata:
  name: e2e-digitalocean
  namespace: kubermatic
spec:
  digitalocean:
    token: ${DO_E2E_TESTS_TOKEN}
EOF
retry 2 kubectl apply -f preset-digitalocean.yaml

echodate "Creating UI GCP preset..."
cat <<EOF > preset-gcp.yaml
apiVersion: kubermatic.k8s.io/v1
kind: Preset
metadata:
  name: e2e-gcp
  namespace: kubermatic
spec:
  gcp:
    serviceAccount: ${GOOGLE_SERVICE_ACCOUNT}
EOF
cat <<EOF > preset-gcp-datacenter.yaml
apiVersion: kubermatic.k8s.io/v1
kind: Preset
metadata:
  name: e2e-gcp-datacenter
  namespace: kubermatic
spec:
  gcp:
    serviceAccount: ${GOOGLE_SERVICE_ACCOUNT}
    datacenter: gcp-westeurope
EOF
retry 2 kubectl apply -f preset-gcp.yaml
retry 2 kubectl apply -f preset-gcp-datacenter.yaml

echodate "Creating UI OpenStack preset..."
cat <<EOF > preset-openstack.yaml
apiVersion: kubermatic.k8s.io/v1
kind: Preset
metadata:
  name: e2e-openstack
  namespace: kubermatic
spec:
  openstack:
    username: ${OS_USERNAME}
    password: ${OS_PASSWORD}
    tenant: ${OS_TENANT_NAME}
    domain: ${OS_DOMAIN}
EOF
retry 2 kubectl apply -f preset-openstack.yaml

echodate "Creating roxy2 user..."
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
retry 2 kubectl apply -f user.yaml

echodate "Running API E2E tests..."
export KUBERMATIC_DEX_VALUES_FILE=$(realpath api/hack/ci/testdata/oauth_values.yaml)
go test -tags="create $KUBERMATIC_EDITION" -timeout 20m ./api/pkg/test/e2e/api -v
go test -tags="e2e $KUBERMATIC_EDITION" -timeout 20m ./api/pkg/test/e2e/api -v
