#!/usr/bin/env bash

set -euo pipefail
export GIT_HEAD_HASH="$(git rev-parse HEAD|tr -d '\n')"
echodate() { echo "$(date) $@"; }
cd $(dirname $0)/../..

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

echodate "Logging into Docker registries"
retry 5 docker login -u $DOCKERHUB_USERNAME -p $DOCKERHUB_PASSWORD
retry 5 docker login -u $QUAY_IO_USERNAME -p $QUAY_IO_PASSWORD
echodate "Successfully logged into all registries"

echodate "Building binaries"
make -C api build
echodate "Successfully finished building binaries"

echodate "Building and pushing docker images"
docker ps &>/dev/null || start-docker.sh
retry 5 ./api/hack/push_image.sh $GIT_HEAD_HASH $(git tag -l --points-at HEAD)
echodate "Sucessfully finished building and pushing docker images"

echodate "Deploying Kubermatic to ci.kubermatic.io"
declare -A chartNamespaces
chartNamespaces=(
["nginx-ingress-controller"]="nginx-ingress-controller"
["cert-manager"]="cert-manager"
["certs"]="default"
["kubermatic"]="kubermatic"
["oauth"]="oauth"
)

for chart in "${!chartNamespaces[@]}"; do
  retry 5 helm upgrade --install --force --wait --timeout 300 --tiller-namespace=kubermatic \
    --namespace=${chartNamespaces[$chart]} \
    --set=kubermatic.isMaster=true \
    --set=kubermatic.controller.image.tag=$GIT_HEAD_HASH \
    --set=kubermatic.api.image.tag=$GIT_HEAD_HASH \
    --set=kubermatic.rbac.image.tag=$GIT_HEAD_HASH \
    $chart ./config/$chart/
done
echodate "Successfully deployed Kubermatic to ci.kubermatic.io"
