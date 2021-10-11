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
source hack/ci/setup-kubermatic-in-kind.sh

echodate "Creating UI Azure preset..."
cat << EOF > preset-azure.yaml
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
cat << EOF > preset-digitalocean.yaml
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
cat << EOF > preset-gcp.yaml
apiVersion: kubermatic.k8s.io/v1
kind: Preset
metadata:
  name: e2e-gcp
  namespace: kubermatic
spec:
  gcp:
    serviceAccount: ${GOOGLE_SERVICE_ACCOUNT}
EOF
cat << EOF > preset-gcp-datacenter.yaml
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
cat << EOF > preset-openstack.yaml
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

echodate "Creating roxy user..."
cat << EOF > user.yaml
apiVersion: kubermatic.k8s.io/v1
kind: User
metadata:
  name: 2d8e8de30d44c46bb0641315ff7b60f05ae92ab0818872f78dc43ebff8ed4614
spec:
  email: roxy@loodse.com
  id: 226655708b35be078a9302f3c79bd7f3982f8cb03ae32872da8e7b23_KUBE
  name: roxy
  admin: false
EOF
retry 2 kubectl apply -f user.yaml

function print_kubermatic_logs {
  if [[ $? -ne 0 ]]; then
    echodate "Printing logs for Kubermatic API"
    kubectl -n kubermatic logs --tail=-1 --selector='app.kubernetes.io/name=kubermatic-api'
    echodate "Printing logs for Master Controller Manager"
    kubectl -n kubermatic logs --tail=-1 --selector='app.kubernetes.io/name=kubermatic-master-controller-manager'
    echodate "Printing logs for Seed Controller Manager"
    kubectl -n kubermatic logs --tail=-1 --selector='app.kubernetes.io/name=kubermatic-seed-controller-manager'
  fi
}
appendTrap print_kubermatic_logs EXIT

echodate "Running API E2E tests..."

go test -tags="create,$KUBERMATIC_EDITION" -timeout 20m ./pkg/test/e2e/api -v
go test -tags="e2e,$KUBERMATIC_EDITION" -timeout 20m ./pkg/test/e2e/api -v
go test -tags="logout,$KUBERMATIC_EDITION" -timeout 20m ./pkg/test/e2e/api -v

echodate "Tests completed successfully!"
