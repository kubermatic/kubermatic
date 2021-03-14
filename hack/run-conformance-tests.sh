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

### Compiles the conformance tests and then runs them in a local Docker
### container (by default). This requires KKP and an OIDC provider (like Dex)
### to be installed, with a `$KUBECONFIG` pointing to the KKP master cluster.
###
### The tests run against a single provider, specified via the `PROVIDER`
### environment variable (default: `aws`). See this script for the
### credential variables for each provider.
###
### OIDC credentials need to be provided either by specifying
### `KUBERMATIC_OIDC_LOGIN` and `KUBERMATIC_OIDC_PASSWORD` environment
### variables or by setting `CREATE_OIDC_TOKEN=false` and setting
### a predefined `KUBEMATIC_OIDC_TOKEN` variable.
###
### Run this script with `-help` to see a list of all available flags on
### the conformance tests. Many of these are set by this script, but you
### can add and override as you like. NB: If test tests run inside a
### container, make sure paths and environment variables can be properly
### resolved.
###
### To disable the Docker container, set the variable `NO_DOCKER=true`.
### In this mode, you need to have the kube-test binaries and all other
### dependencies installed locally on your machine, but it makes testing
### against a local KKP setup much easier.

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

if [ -z "${VAULT_ADDR:-}" ]; then
  export VAULT_ADDR=https://vault.loodse.com/
fi

if [ -z "${KUBECONFIG:-}" ]; then
  echodate "No \$KUBECONFIG set, defaulting to Vault key dev/seed-clusters/dev.kubermatic.io"

  KUBECONFIG="$(mktemp)"
  vault kv get -field=kubeconfig dev/seed-clusters/dev.kubermatic.io > $KUBECONFIG
fi

extraArgs=""
provider="${PROVIDER:-aws}"

case "$provider" in
alibaba)
  extraArgs="-alibaba-access-key-id=$ALIBABA_KEY_ID
      -alibaba-secret-access-key=$ALIBABA_SECRET"
  ;;

aws)
  AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-$(vault kv get -field=accessKeyID dev/e2e-aws)}"
  AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-$(vault kv get -field=secretAccessKey dev/e2e-aws)}"
  extraArgs="-aws-access-key-id=$AWS_ACCESS_KEY_ID
      -aws-secret-access-key=$AWS_SECRET_ACCESS_KEY"
  ;;

azure)
  AZURE_CLIENT_ID="${AZURE_CLIENT_ID:-$(vault kv get -field=clientID dev/e2e-azure)}"
  AZURE_CLIENT_SECRET="${AZURE_CLIENT_SECRET:-$(vault kv get -field=clientSecret dev/e2e-azure)}"
  AZURE_TENANT_ID="${AZURE_TENANT_ID:-$(vault kv get -field=tenantID dev/e2e-azure)}"
  AZURE_SUBSCRIPTION_ID="${AZURE_SUBSCRIPTION_ID:-$(vault kv get -field=subscriptionID dev/e2e-azure)}"
  extraArgs="-azure-client-id=$AZURE_CLIENT_ID
      -azure-client-secret=$AZURE_CLIENT_SECRET
      -azure-tenant-id=$AZURE_TENANT_ID
      -azure-subscription-id=$AZURE_SUBSCRIPTION_ID"
  ;;

digitalocean)
  DO_TOKEN="${DO_TOKEN:-$(vault kv get -field=token dev/e2e-digitalocean)}"
  extraArgs="-digitalocean-token=$DO_TOKEN"
  ;;

gcp)
  extraArgs="-gcp-service-account=$GOOGLE_SERVICE_ACCOUNT"
  ;;

hetzner)
  HZ_TOKEN="${HZ_TOKEN:-$(vault kv get -field=token dev/e2e-hetzner)}"
  extraArgs="-hetzner-token=$HZ_TOKEN"
  ;;

kubevirt)
  extraArgs="-kubevirt-kubeconfig=${KUBEVIRT_KUBECONFIG}"
  ;;

openstack)
  OS_DOMAIN="${OS_DOMAIN:-$(vault kv get -field=OS_USER_DOMAIN_NAME dev/syseleven-openstack)}"
  OS_TENANT_NAME="${OS_TENANT_NAME:-$(vault kv get -field=OS_TENANT_NAME dev/syseleven-openstack)}"
  OS_USERNAME="${OS_USERNAME:-$(vault kv get -field=username dev/syseleven-openstack)}"
  OS_PASSWORD="${OS_PASSWORD:-$(vault kv get -field=password dev/syseleven-openstack)}"
  OS_APPLICATION_CREDENTIAL_ID="${OS_APPLICATION_CREDENTIAL_ID:-$(vault kv get -field=application-credential-id dev/syseleven-openstack)}"
  OS_APPLICATION_CREDENTIAL_SECRET="${OS_APPLICATION_CREDENTIAL_SECRET:-$(vault kv get -field=application-credential-secret dev/syseleven-openstack)}"
  extraArgs="-openstack-domain=$OS_DOMAIN
      -openstack-tenant=$OS_TENANT_NAME
      -openstack-username=$OS_USERNAME
      -openstack-password=$OS_PASSWORD
      -openstack-application-credential-id=$OS_APPLICATION_CREDENTIAL_ID
      -openstack-application-credential-secret=$OS_APPLICATION_CREDENTIAL_SECRET"
  ;;

packet)
  extraArgs="-packet-api-key=$PACKET_API_KEY
      -packet-project-id=$PACKET_PROJECT_ID"
  ;;

vsphere)
  VSPHERE_USERNAME="${VSPHERE_USERNAME:-$(vault kv get -field=username dev/vsphere)}"
  VSPHERE_PASSWORD="${VSPHERE_PASSWORD:-$(vault kv get -field=password dev/vsphere)}"
  extraArgs="-vsphere-username=$VSPHERE_USERNAME
      -vsphere-password=$VSPHERE_PASSWORD"
  ;;

*)
  echodate "Unknown provider $provider (\$PROVIDER) selected."
  exit 1
  ;;
esac

if [ -n "${VERSIONS:-}" ]; then
  extraArgs="$extraArgs -versions=$VERSIONS"
fi

if [ -n "${SEED_NAME:-}" ]; then
  extraArgs="$extraArgs -kubermatic-seed-cluster=$SEED_NAME"
fi

endpoint="${KUBERMATIC_API_ENDPOINT:-https://dev.kubermatic.io}"
oidcToken="${KUBEMATIC_OIDC_TOKEN:-}"

# allow to transport additional env variables into the container
# to not reveal credentials as CLI flags; set a dummy value to
# keep the `docker run` command easier to write
if [ -z "${EXTRA_ENV:-}" ]; then
  EXTRA_ENV="KUBERMATIC=1"
fi

if [ -f ~/.ssh/id_rsa.pub ]; then
  extraArgs="$extraArgs -node-ssh-pub-key=/usrhome/.ssh/id_rsa.pub"
else
  # explicitly disable the auto-include, which would fail inside the container
  extraArgs="$extraArgs -node-ssh-pub-key="
fi

mkdir -p reports

if [ -n "${NO_DOCKER:-}" ]; then
  echodate "Compiling conformance-tests..."
  make conformance-tests

  echodate "Starting conformance-tests..."
  _build/conformance-tests $extraArgs \
    -log-format=console \
    -worker-name="$USER" \
    -kubeconfig=$KUBECONFIG \
    -reports-root=reports \
    -kubermatic-endpoint="$endpoint" \
    -kubermatic-oidc-token="$oidcToken" \
    -kubermatic-delete-cluster=true \
    -providers="$provider" \
    -distributions="flatcar" \
    $@
else
  echodate "Compiling conformance-tests..."
  # make sure to compile a conformance-tester binary that can actually
  # run inside the container
  GOOS=linux GOARCH=amd64 make conformance-tests

  echodate "Starting conformance-tests in Docker..."
  docker run \
    --rm \
    --interactive \
    --tty \
    --volume $PWD/reports:/reports \
    --volume $PWD:/go/src/k8c.io/kubermatic \
    --volume "$(realpath "$KUBECONFIG"):/kubeconfig" \
    --volume $HOME:/usrhome \
    --workdir /go/src/k8c.io/kubermatic \
    --user "$(id -u):$(id -g)" \
    --env "KUBERMATIC_OIDC_LOGIN=${KUBERMATIC_OIDC_LOGIN-}" \
    --env "KUBERMATIC_OIDC_PASSWORD=${KUBERMATIC_OIDC_PASSWORD-}" \
    --env "${EXTRA_ENV:-}" \
    quay.io/kubermatic/e2e-kind:with-conformance-tests-v1.0.24 \
    _build/conformance-tests $extraArgs \
    -log-format=console \
    -worker-name="$USER" \
    -kubeconfig=/kubeconfig \
    -reports-root=/reports \
    -kubermatic-endpoint="$endpoint" \
    -kubermatic-oidc-token="$oidcToken" \
    -kubermatic-delete-cluster=true \
    -providers="$provider" \
    -distributions="flatcar" \
    $@
fi
