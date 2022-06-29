#!/usr/bin/env bash

# Copyright 2020 The Kubermatic Kubernetes Platform contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

### This script sets up a local KKP installation in kind, deploys a
### couple of test Presets and Users and then runs the e2e tests for the
### API.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

export KIND_CLUSTER_NAME="${SEED_NAME:-kubermatic}"

source hack/ci/setup-kind-cluster.sh

# gather the logs of all things in the Kubermatic namespace
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &

source hack/ci/setup-kubermatic-in-kind.sh

echodate "Creating Azure preset..."
cat << EOF > preset-azure.yaml
apiVersion: kubermatic.k8c.io/v1
kind: Preset
metadata:
  name: e2e-azure
  namespace: kubermatic
spec:
  azure:
    tenantID: ${AZURE_E2E_TESTS_TENANT_ID}
    subscriptionID: ${AZURE_E2E_TESTS_SUBSCRIPTION_ID}
    clientID: ${AZURE_E2E_TESTS_CLIENT_ID}
    clientSecret: ${AZURE_E2E_TESTS_CLIENT_SECRET}
    loadBalancerSKU: "standard"
EOF
retry 2 kubectl apply -f preset-azure.yaml

echodate "Creating Hetzner preset..."
cat << EOF > preset-hetzner.yaml
apiVersion: kubermatic.k8c.io/v1
kind: Preset
metadata:
  name: e2e-hetzner
  namespace: kubermatic
spec:
  hetzner:
    token: ${HZ_E2E_TOKEN}
EOF
retry 2 kubectl apply -f preset-hetzner.yaml

echodate "Creating DigitalOcean preset..."
cat << EOF > preset-digitalocean.yaml
apiVersion: kubermatic.k8c.io/v1
kind: Preset
metadata:
  name: e2e-digitalocean
  namespace: kubermatic
spec:
  digitalocean:
    token: ${DO_E2E_TESTS_TOKEN}
EOF
retry 2 kubectl apply -f preset-digitalocean.yaml

echodate "Creating GCP preset..."
cat << EOF > preset-gcp.yaml
apiVersion: kubermatic.k8c.io/v1
kind: Preset
metadata:
  name: e2e-gcp
  namespace: kubermatic
spec:
  gcp:
    serviceAccount: ${GOOGLE_SERVICE_ACCOUNT}
EOF
cat << EOF > preset-gcp-datacenter.yaml
apiVersion: kubermatic.k8c.io/v1
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

echodate "Creating OpenStack preset..."
cat << EOF > preset-openstack.yaml
apiVersion: kubermatic.k8c.io/v1
kind: Preset
metadata:
  name: e2e-openstack
  namespace: kubermatic
spec:
  openstack:
    username: ${OS_USERNAME}
    password: ${OS_PASSWORD}
    project: ${OS_TENANT_NAME}
    projectID: ""
    domain: ${OS_DOMAIN}
EOF
retry 2 kubectl apply -f preset-openstack.yaml

echodate "Creating roxy-admin user..."
cat << EOF > user.yaml
apiVersion: kubermatic.k8c.io/v1
kind: User
metadata:
  name: roxy-admin
spec:
  admin: true
  email: roxy-admin@kubermatic.com
  name: roxy-admin
EOF
retry 2 kubectl apply -f user.yaml

echodate "Running API E2E tests..."

go_test kubermatic_api_create -tags="create,$KUBERMATIC_EDITION" -timeout 20m ./pkg/test/e2e/api -v
go_test kubermatic_api_e2e -tags="e2e,$KUBERMATIC_EDITION" -timeout 20m ./pkg/test/e2e/api -v
go_test kubermatic_api_logout -tags="logout,$KUBERMATIC_EDITION" -timeout 20m ./pkg/test/e2e/api -v

echodate "Tests completed successfully!"
