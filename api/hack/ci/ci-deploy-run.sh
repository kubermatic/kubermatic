#!/usr/bin/env bash

set -euo pipefail
export GIT_HEAD_HASH="$(git rev-parse HEAD|tr -d '\n')"
echodate() { echo "$(date) $@"; }
cd $(dirname $0)/../..

if [ "$#" -lt 1 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename $0) (master|seed) path/to/${VALUES_FILE}
EOF
  exit 0
fi

if [[ ! -f ${2} ]]; then
    echo "File not found!"
    exit 1
fi

VALUES_FILE=$(realpath ${2})

function retry {
  local retries=$1
  shift

  local count=0
  until "$@"; do
    exit=$?
    wait=$((2 ** $count))
    count=$(($count + 1))
    if [ $count -lt $retries ]; then
      echo "Retry $count/$retries exited $exit, retrying in $wait seconds..."
      sleep $wait
    else
      echo "Retry $count/$retries exited $exit, no more retries left."
      return $exit
    fi
  done
  return 0
}

declare -A chartNamespaces
chartNamespaces=(
["nginx-ingress-controller"]="nginx-ingress-controller"
["cert-manager"]="cert-manager"
["certs"]="default"
["minio"]="minio"
["nodeport-proxy"]="nodeport-proxy"
["iap"]="iap"
["kubermatic"]="kubermatic"
["oauth"]="oauth"
["prometheus"]="monitoring"
["node-exporter"]="monitoring"
["kube-state-metrics"]="monitoring"
["grafana"]="monitoring"
["alertmanager"]="monitoring"
["elasticsearch"]="logging"
["fluentbit"]="logging"
["kibana"]="logging"
)

function deploy {
  retry 5 kubectl apply -f ./config/kubermatic/crd/
  for chart in "${!chartNamespaces[@]}"; do
    if [[ "${chartNamespaces[$chart]}" == "monitoring" ]] || [[ "${chartNamespaces[$chart]}" == "logging" ]] ; then
      directory_name="${chartNamespaces[$chart]}/$chart"
    else
      directory_name=$chart
    fi

    retry 5 helm $1 upgrade --install --force --wait --timeout 300 \
      --namespace=${chartNamespaces[$chart]} \
      --values $VALUES_FILE \
      --set=kubermatic.controller.image.tag=$GIT_HEAD_HASH \
      --set=kubermatic.api.image.tag=$GIT_HEAD_HASH \
      --set=kubermatic.rbac.image.tag=$GIT_HEAD_HASH \
      $chart ./config/$directory_name/
  done
}

echodate "Logging into Quay"
docker ps &>/dev/null || start-docker.sh
retry 5 docker login -u $QUAY_IO_USERNAME -p $QUAY_IO_PASSWORD quay.io
echodate "Successfully logged into Quay"

echodate "Building binaries"
time make -C api build
echodate "Successfully finished building binaries"

echodate "Building and pushing quay images"
retry 5 ./api/hack/push_image.sh $GIT_HEAD_HASH $(git tag -l --points-at HEAD)
echodate "Sucessfully finished building and pushing quay images"

echodate "Getting secrets from Vault"
export VAULT_ADDR=https://vault.loodse.com/
retry 5 vault write \
  --format=json auth/approle/login \
  role_id=$VAULT_ROLE_ID secret_id=$VAULT_SECRET_ID > /tmp/vault-token-response.json
export VAULT_TOKEN="$(cat /tmp/vault-token-response.json| jq .auth.client_token -r)"
export KUBECONFIG=/tmp/kubeconfig
export VALUES_FILE=/tmp/values.yaml

# deploy to drun
vault kv get -field=kubeconfig \
  dev/seed-clusters/run.kubermatic.io > $KUBECONFIG
vault kv get -field=values.yaml \
  dev/seed-clusters/run.kubermatic.io > $VALUES_FILE
echodate "Successfully got secrets for run from Vault"

echodate "Deploying Kubermatic to run.kubermatic.io"
deploy
echodate "Successfully deployed Kubermatic to run.kubermatic.io"
