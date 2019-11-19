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

make build
docker build -t quay.io/kubermatic/api:${1} .
cd cmd/nodeport-proxy && export TAG=${1} && make docker && unset TAG && cd -
cd cmd/kubeletdnat-controller && export TAG=${1} && make docker && unset TAG && cd -
docker build -t "quay.io/kubermatic/addons:${1}" ../addons
docker build -t "quay.io/kubermatic/openshift-addons:${1}" ../openshift_addons
docker build -t "quay.io/kubermatic/user-ssh-keys-agent:${1}" ./cmd/user-ssh-keys-agent

for TAG in "$@"
do
    if [[ -z "$TAG" ]]; then
      continue
    fi

    echo "Tagging ${TAG}"
    docker tag quay.io/kubermatic/api:${1} quay.io/kubermatic/api:${TAG}
    docker tag quay.io/kubermatic/nodeport-proxy:${1} quay.io/kubermatic/nodeport-proxy:${TAG}
    docker tag quay.io/kubermatic/kubeletdnat-controller:${1}  quay.io/kubermatic/kubeletdnat-controller:${TAG}
    docker tag quay.io/kubermatic/addons:${1} quay.io/kubermatic/addons:${TAG}
    docker tag quay.io/kubermatic/openshift-addons:${1} quay.io/kubermatic/openshift-addons:${TAG}
    docker tag quay.io/kubermatic/user-ssh-keys-agent:${1} quay.io/kubermatic/user-ssh-keys-agent:${TAG}

    docker push quay.io/kubermatic/api:${TAG}
    docker push quay.io/kubermatic/nodeport-proxy:${TAG}
    docker push quay.io/kubermatic/kubeletdnat-controller:${TAG}
    docker push quay.io/kubermatic/addons:${TAG}
    docker push quay.io/kubermatic/openshift-addons:${TAG}
    docker push quay.io/kubermatic/user-ssh-keys-agent:${TAG}
done
