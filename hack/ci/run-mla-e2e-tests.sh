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

### This script sets up a local KKP installation in kind, deploys a
### couple of test Presets and Users and then runs the MLA e2e tests.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

if [ -z "${E2E_SSH_PUBKEY:-}" ]; then
  echodate "Getting default SSH pubkey for machines from Vault"
  retry 5 vault_ci_login
  E2E_SSH_PUBKEY="$(mktemp)"
  vault kv get -field=pubkey dev/e2e-machine-controller-ssh-key > "${E2E_SSH_PUBKEY}"
else
  E2E_SSH_PUBKEY_CONTENT="${E2E_SSH_PUBKEY}"
  E2E_SSH_PUBKEY="$(mktemp)"
  echo "${E2E_SSH_PUBKEY_CONTENT}" > "${E2E_SSH_PUBKEY}"
fi

echodate "SSH public key will be $(head -c 25 ${E2E_SSH_PUBKEY})...$(tail -c 25 ${E2E_SSH_PUBKEY})"

export KIND_CLUSTER_NAME="${SEED_NAME:-kubermatic}"
export KUBERMATIC_YAML=hack/ci/testdata/kubermatic.yaml

source hack/ci/setup-kind-cluster.sh

# gather the logs of all things in the cluster control plane and in the Kubermatic namespace
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/cluster-control-plane" --namespace 'cluster-*' > /dev/null 2>&1 &
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &

source hack/ci/setup-kubermatic-mla-in-kind.sh

echodate "Waiting for kubermatic-webhook deployment to exist..."
timeout=300
while [ $timeout -gt 0 ]; do
  if kubectl -n kubermatic get deployment kubermatic-webhook &> /dev/null; then
    echodate "kubermatic-webhook deployment found"
    break
  fi
  sleep 2
  timeout=$((timeout - 2))
done

if [ $timeout -le 0 ]; then
  echodate "ERROR: kubermatic-webhook deployment not found after 5 minutes"
  kubectl -n kubermatic get deployments
  exit 1
fi

echodate "Waiting for kubermatic-webhook pods to be ready..."
retry 30 kubectl -n kubermatic wait --for=condition=ready --timeout=10s pod -l app.kubernetes.io/name=kubermatic-webhook || {
  echodate "WARNING: kubermatic-webhook pods not ready yet, checking status..."
  kubectl -n kubermatic get pods -l app.kubernetes.io/name=kubermatic-webhook
  kubectl -n kubermatic describe pods -l app.kubernetes.io/name=kubermatic-webhook
}

echodate "Waiting for kubermatic-webhook service endpoint to be ready..."
timeout=120
while [ $timeout -gt 0 ]; do
  if kubectl -n kubermatic get endpoints kubermatic-webhook -o jsonpath='{.subsets[*].addresses[*].ip}' | grep -q .; then
    echodate "kubermatic-webhook service has endpoints"
    break
  fi
  sleep 2
  timeout=$((timeout - 2))
done

echodate "Waiting for User CRD to be ready..."
retry 5 kubectl get crd users.kubermatic.k8c.io

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

echodate "Checking if user exists..."
if kubectl get user roxy-admin &> /dev/null; then
  echodate "User roxy-admin already exists, checking email..."
fi

kubectl get user

echodate "Running MLA tests..."

go_test mla_e2e -timeout 30m -tags mla -v ./pkg/test/e2e/mla \
  -kubeconfig "$KUBECONFIG" \
  -aws-kkp-datacenter "$AWS_E2E_TESTS_DATACENTER" \
  -ssh-pub-key "$(cat "$E2E_SSH_PUBKEY")"

echodate "Tests completed successfully!"
