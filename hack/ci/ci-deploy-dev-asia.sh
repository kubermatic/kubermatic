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
export USE_KUBERMATIC_OPERATOR=true

# deploy to dev-asia
vault kv get -field=kubeconfig dev/seed-clusters/dev.kubermatic.io > ${KUBECONFIG}
vault kv get -field=asia-south1-c-values.yaml dev/seed-clusters/dev.kubermatic.io > ${VALUES_FILE}
kubectl config use-context asia-south1-c
echodate "Successfully got secrets for dev-asia from Vault"

echodate "Deploying ${DEPLOY_STACK} stack to dev-asia"
TILLER_NAMESPACE=kubermatic-installer ./api/hack/deploy.sh seed ${VALUES_FILE}
echodate "Successfully deployed ${DEPLOY_STACK} stack to dev-asia"
