#!/usr/bin/env bash

set -euo pipefail
export DEPLOY_STACK=${DEPLOY_STACK:-kubermatic}
export GIT_HEAD_HASH="$(git rev-parse HEAD|tr -d '\n')"
cd $(dirname $0)/../../..

source ./api/hack/lib.sh

if [[ "${DEPLOY_STACK}" == "kubermatic" ]]; then
  ./api/hack/ci/ci-push-images.sh
fi

echodate "Getting secrets from Vault"
export VAULT_ADDR=https://vault.loodse.com/
retry 5 vault write \
  --format=json auth/approle/login \
  role_id=${VAULT_ROLE_ID} secret_id=${VAULT_SECRET_ID} > /tmp/vault-token-response.json
export VAULT_TOKEN="$(cat /tmp/vault-token-response.json| jq .auth.client_token -r)"
export KUBECONFIG=/tmp/kubeconfig
export VALUES_FILE=/tmp/values.yaml
export HELM_EXTRA_ARGS="--set=kubermatic.controller.image.tag=${GIT_HEAD_HASH} \
    --set=kubermatic.api.image.tag=${GIT_HEAD_HASH} \
    --set=kubermatic.masterController.image.tag=${GIT_HEAD_HASH} \
    --set=kubermatic.controller.addons.kubernetes.image.tag=${GIT_HEAD_HASH}"

# deploy to cloud-asia
vault kv get -field=kubeconfig dev/seed-clusters/cloud.kubermatic.io > ${KUBECONFIG}
vault kv get -field=asia-east1-a-1-values.yaml dev/seed-clusters/cloud.kubermatic.io > ${VALUES_FILE}
kubectl config use-context asia-east1-a-1
echodate "Successfully got secrets for cloud-asia from Vault"

echodate "Deploying ${DEPLOY_STACK} stack to cloud-asia"
TILLER_NAMESPACE=kubermatic-installer ./api/hack/deploy.sh seed ${VALUES_FILE} ${HELM_EXTRA_ARGS}
echodate "Successfully deployed ${DEPLOY_STACK} stack to cloud-asia"
