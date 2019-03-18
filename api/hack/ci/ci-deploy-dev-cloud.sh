#!/usr/bin/env bash

set -euo pipefail
export GIT_HEAD_HASH="$(git rev-parse HEAD|tr -d '\n')"
cd $(dirname $0)/../../..

source ./api/hack/lib.sh

echodate "Logging into Quay"
docker ps &>/dev/null || start-docker.sh
retry 5 docker login -u ${QUAY_IO_USERNAME} -p ${QUAY_IO_PASSWORD} quay.io
echodate "Successfully logged into Quay"

echodate "Building binaries"
time make -C api build
echodate "Successfully finished building binaries"

echodate "Building and pushing quay images"
retry 5 ./api/hack/push_image.sh $GIT_HEAD_HASH $(git tag -l --points-at HEAD) "latest"
echodate "Sucessfully finished building and pushing quay images"

echodate "Getting secrets from Vault"
export VAULT_ADDR=https://vault.loodse.com/
retry 5 vault write \
  --format=json auth/approle/login \
  role_id=${VAULT_ROLE_ID} secret_id=${VAULT_SECRET_ID} > /tmp/vault-token-response.json
export VAULT_TOKEN="$(cat /tmp/vault-token-response.json| jq .auth.client_token -r)"
export KUBECONFIG=/tmp/kubeconfig
export VALUES_FILE=/tmp/values.yaml
export HELM_EXTRA_ARGS="--tiller-namespace=kubermatic-installer \
    --set=kubermatic.controller.image.tag=${GIT_HEAD_HASH} \
    --set=kubermatic.api.image.tag=${GIT_HEAD_HASH} \
    --set=kubermatic.rbac.image.tag=${GIT_HEAD_HASH}"

# deploy to dev
vault kv get -field=kubeconfig dev/seed-clusters/dev.kubermatic.io > ${KUBECONFIG}
vault kv get -field=values.yaml dev/seed-clusters/dev.kubermatic.io > ${VALUES_FILE}
echodate "Successfully got secrets for dev from Vault"

echodate "Deploying Kubermatic to dev"
./api/hack/deploy.sh master ${VALUES_FILE} ${HELM_EXTRA_ARGS}

# deploy to cloud-asia
vault kv get -field=kubeconfig dev/seed-clusters/cloud.kubermatic.io > ${KUBECONFIG}
vault kv get -field=asia-east1-a-1-values.yaml dev/seed-clusters/cloud.kubermatic.io > ${VALUES_FILE}
kubectl config use-context asia-east1-a-1
echodate "Successfully got secrets for cloud-asia from Vault"

echodate "Deploying Kubermatic to cloud-asia"
./api/hack/deploy.sh seed ${VALUES_FILE} ${HELM_EXTRA_ARGS}

# deploy to cloud-eu
vault kv get -field=kubeconfig dev/seed-clusters/cloud.kubermatic.io > ${KUBECONFIG}
vault kv get -field=europe-west3-c-1-values.yaml dev/seed-clusters/cloud.kubermatic.io > ${VALUES_FILE}
kubectl config use-context europe-west3-c-1
echodate "Successfully got secrets for cloud-eu from Vault"

echodate "Deploying Kubermatic to cloud-eu"
./api/hack/deploy.sh master ${VALUES_FILE} ${HELM_EXTRA_ARGS}

# deploy to cloud-us
vault kv get -field=kubeconfig dev/seed-clusters/cloud.kubermatic.io > ${KUBECONFIG}
vault kv get -field=us-central1-c-1-values.yaml dev/seed-clusters/cloud.kubermatic.io > ${VALUES_FILE}
kubectl config use-context us-central1-c-1
echodate "Successfully got secrets for cloud-us from Vault"

echodate "Deploying Kubermatic to cloud-us"
./api/hack/deploy.sh seed ${VALUES_FILE} ${HELM_EXTRA_ARGS}
