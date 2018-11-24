#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename $0) tag1 tag2

Example:
  $(basename $0) 0cf36a568b0911ac6688115df53c1f1701f45fcd6be5fc97fd6dc0410ac37a82 v2.5
EOF
  exit 0
fi

cd "$(dirname "$0")/../"

docker login -u $(vault kv get -field=username dev/kubermatic-docker) -p $(vault kv get -field=password dev/kubermatic-docker)
docker login -u $(vault kv get -field=username dev/kubermatic-quay.io) -p $(vault kv get -field=password dev/kubermatic-quay.io) quay.io

docker build -t docker.io/kubermatic/api:${1} .

for TAG in "$@"
do
    if [[ -z "$TAG" ]]; then
      continue
    fi

    echo "Tagging ${TAG}"
    # Docker Hub
    docker tag docker.io/kubermatic/api:${1} docker.io/kubermatic/api:${TAG}
    docker push docker.io/kubermatic/api:${TAG}
    # Quay.io
    docker tag docker.io/kubermatic/api:${1} quay.io/kubermatic/api:${TAG}
    docker push quay.io/kubermatic/api:${TAG}
done
