#!/usr/bin/env bash

# Copyright 2021 The Kubermatic Kubernetes Platform contributors.
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

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

function cleanup() {
  if [[ -n "${TMP:-}" ]]; then
    rm -rf "${TMP}"
  fi
}
trap cleanup EXIT SIGINT SIGTERM

export VM_IMAGE_PATH="${VM_IMAGE_PATH:-$HOME}"
export VM_NAME="${VM_NAME:-CentOS-7-x86_64-GenericCloud.qcow2}"
export KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kubermatic}"
export KUBERMATIC_OIDC_LOGIN=roxy@loodse.com
export KUBERMATIC_OIDC_PASSWORD=password
export KUBERMATIC_DEX_VALUES_FILE=$(realpath hack/ci/testdata/oauth_values.yaml)
export KUBERMATIC_API_ENDPOINT=http://localhost:8080
export KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ce}"
export DATA_FILE=$(realpath hack/local/data)

if [ -z "${VAULT_ADDR:-}" ]; then
  export VAULT_ADDR=https://vault.kubermatic.com/
fi

GOOGLE_SERVICE_ACCOUNT="${GOOGLE_SERVICE_ACCOUNT:-$(vault kv get -field=serviceAccount dev/e2e-gce)}"

DO_E2E_TESTS_TOKEN="${DO_E2E_TESTS_TOKEN:-$(vault kv get -field=token dev/e2e-digitalocean)}"

AZURE_E2E_TESTS_TENANT_ID="${AZURE_E2E_TESTS_TENANT_ID:-$(vault kv get -field=tenantID dev/e2e-azure)}"
AZURE_E2E_TESTS_SUBSCRIPTION_ID="${AZURE_E2E_TESTS_SUBSCRIPTION_ID:-$(vault kv get -field=subscriptionID dev/e2e-azure)}"
AZURE_E2E_TESTS_CLIENT_ID="${AZURE_E2E_TESTS_CLIENT_ID:-$(vault kv get -field=clientID dev/e2e-azure)}"
AZURE_E2E_TESTS_CLIENT_SECRET="${AZURE_E2E_TESTS_CLIENT_SECRET:-$(vault kv get -field=clientSecret dev/e2e-azure)}"

OS_USERNAME="${OS_USERNAME:-$(vault kv get -field=username dev/e2e-openstack)}"
OS_PASSWORD="${OS_PASSWORD:-$(vault kv get -field=password dev/e2e-openstack)}"
OS_TENANT_NAME="${OS_TENANT_NAME:-$(vault kv get -field=tenant dev/e2e-openstack)}"
OS_DOMAIN="${OS_DOMAIN:-$(vault kv get -field=domain dev/e2e-openstack)}"

export KUBECONFIG=~/.kube/config

TMP=$(mktemp -d)

echodate "Creating roxy2 user..."
cat << EOF > "$TMP"/user.yaml
apiVersion: kubermatic.k8s.io/v1
kind: User
metadata:
  name: c41724e256445bf133d6af1168c2d96a7533cd437618fdbe6dc2ef1fee97acd3
spec:
  admin: true
  email: roxy2@loodse.com
  id: 1413636a43ddc27da27e47614faedff24b4ab19c9d9f2b45dd1b89d9_KUBE
  name: roxy2
EOF
retry 2 kubectl apply -f "$TMP"/user.yaml

echodate "Creating UI Azure preset..."
cat << EOF > "$TMP"/preset-azure.yaml
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
retry 2 kubectl apply -f "$TMP"/preset-azure.yaml

echodate "Creating UI DigitalOcean preset..."
cat << EOF > "$TMP"/preset-digitalocean.yaml
apiVersion: kubermatic.k8s.io/v1
kind: Preset
metadata:
  name: e2e-digitalocean
  namespace: kubermatic
spec:
  digitalocean:
    token: ${DO_E2E_TESTS_TOKEN}
EOF
retry 2 kubectl apply -f "$TMP"/preset-digitalocean.yaml

echodate "Creating UI GCP preset..."
cat << EOF > "$TMP"/preset-gcp.yaml
apiVersion: kubermatic.k8s.io/v1
kind: Preset
metadata:
  name: e2e-gcp
  namespace: kubermatic
spec:
  gcp:
    serviceAccount: ${GOOGLE_SERVICE_ACCOUNT}
EOF

cat << EOF > "$TMP"/preset-gcp-datacenter.yaml
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

retry 2 kubectl apply -f "$TMP"/preset-gcp.yaml
retry 2 kubectl apply -f "$TMP"/preset-gcp-datacenter.yaml

echodate "Creating UI OpenStack preset..."
cat << EOF > "$TMP"/preset-openstack.yaml
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
retry 2 kubectl apply -f "$TMP"/preset-openstack.yaml

echodate "Creating UI KubeVirt preset..."
ENCODED_KUBECONFIG=$(kind get kubeconfig --name ${KIND_CLUSTER_NAME} --internal | base64 -w0)
cat << EOF > "$TMP"/preset-kubevirt.yaml
apiVersion: kubermatic.k8s.io/v1
kind: Preset
metadata:
  name: e2e-kubevirt
  namespace: kubermatic
spec:
  kubevirt:
    kubeconfig: ${ENCODED_KUBECONFIG}
EOF
retry 2 kubectl apply -f "$TMP"/preset-kubevirt.yaml

echodate "Installing KubeVirt"
kubectl apply -f https://github.com/kubevirt/kubevirt/releases/download/v0.45.0/kubevirt-operator.yaml
kubectl apply -f https://github.com/kubevirt/kubevirt/releases/download/v0.45.0/kubevirt-cr.yaml
kubectl apply -f https://github.com/kubevirt/containerized-data-importer/releases/download/v1.40.0/cdi-operator.yaml
kubectl apply -f https://github.com/kubevirt/containerized-data-importer/releases/download/v1.40.0/cdi-cr.yaml
echodate "Waiting for load balancer to be ready..."
retry 10 check_all_deployments_ready kubevirt
echodate "KubeVirt is ready."

echodate "Installing local repo for VMs"
retry 2 kubectl apply -f "$DATA_FILE"/vm-repo.yaml
retry 8 check_all_deployments_ready default

if [ ! -f "$VM_IMAGE_PATH"/"$VM_NAME" ]; then
  echodate "Getting $VM_NAME image"
  curl http://cloud.centos.org/centos/7/images/CentOS-7-x86_64-GenericCloud.qcow2 -o "$VM_IMAGE_PATH"/"$VM_NAME"
fi

VM_POD=$(kubectl get pod -l app=vm-repo --output=jsonpath={.items..metadata.name})
retry 2 kubectl cp "$VM_IMAGE_PATH"/"$VM_NAME" "$VM_POD":/usr/share/nginx/html

if [ -z $(kubectl get service vm-repo -o=name --ignore-not-found) ]; then
  echodate "Creating vm-repo service"
  retry 2 kubectl expose deployment vm-repo
fi

echodate "Running API E2E tests..."
go test -tags="kubevirt" -timeout 20m ./pkg/test/e2e/api -v
go test -tags="e2e,$KUBERMATIC_EDITION" -timeout 20m ./pkg/test/e2e/api -v
go test -tags="logout,$KUBERMATIC_EDITION" -timeout 20m ./pkg/test/e2e/api -v
go clean -testcache
echodate "Tests completed successfully!"
