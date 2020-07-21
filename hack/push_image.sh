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

set -euo pipefail

if [ "$#" -lt 1 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename $0) tag1 tag2

Example:
  $(basename $0) 0cf36a568b0911ac6688115df53c1f1701f45fcd6be5fc97fd6dc0410ac37a82 v2.5
EOF
  exit 0
fi

cd $(dirname "$0")/..
source hack/lib.sh

DOCKER_REPO="${DOCKER_REPO:-quay.io/kubermatic}"

REPOSUFFIX=""
KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ee}"

if [ "$KUBERMATIC_EDITION" != "ce" ]; then
  REPOSUFFIX="-$KUBERMATIC_EDITION"
fi

make build
docker build -t ${DOCKER_REPO}/kubermatic${REPOSUFFIX}:${1} .
cd cmd/nodeport-proxy && export TAG=${1} && make DOCKER_REPO=${DOCKER_REPO} docker && unset TAG && cd -
cd cmd/kubeletdnat-controller && export TAG=${1} && make DOCKER_REPO=${DOCKER_REPO} docker && unset TAG && cd -
docker build -t "${DOCKER_REPO}/addons:${1}" addons
docker build -t "${DOCKER_REPO}/openshift-addons:${1}" openshift_addons
cd cmd/user-ssh-keys-agent && export TAG=${1} && make DOCKER_REPO=${DOCKER_REPO} docker && unset TAG && cd -
docker build  -t ${DOCKER_REPO}/etcd-launcher:${1} -f cmd/etcd-launcher/Dockerfile .

# keep a mirror of the EE version in the old repo
if [ "$KUBERMATIC_EDITION" == "ee" ]; then
  docker tag ${DOCKER_REPO}/kubermatic${REPOSUFFIX}:${1} ${DOCKER_REPO}/api:${1}
fi

for TAG in "$@"; do
  if [[ -z "$TAG" ]]; then
    continue
  fi

  echodate "Tagging ${TAG}"
  docker tag ${DOCKER_REPO}/kubermatic${REPOSUFFIX}:${1} ${DOCKER_REPO}/kubermatic${REPOSUFFIX}:${TAG}
  docker tag ${DOCKER_REPO}/nodeport-proxy:${1} ${DOCKER_REPO}/nodeport-proxy:${TAG}
  docker tag ${DOCKER_REPO}/kubeletdnat-controller:${1}  ${DOCKER_REPO}/kubeletdnat-controller:${TAG}
  docker tag ${DOCKER_REPO}/addons:${1} ${DOCKER_REPO}/addons:${TAG}
  docker tag ${DOCKER_REPO}/openshift-addons:${1} ${DOCKER_REPO}/openshift-addons:${TAG}
  docker tag ${DOCKER_REPO}/user-ssh-keys-agent:${1} ${DOCKER_REPO}/user-ssh-keys-agent:${TAG}
  docker tag ${DOCKER_REPO}/etcd-launcher:${1} ${DOCKER_REPO}/etcd-launcher:${TAG}

  docker push ${DOCKER_REPO}/kubermatic${REPOSUFFIX}:${TAG}
  docker push ${DOCKER_REPO}/nodeport-proxy:${TAG}
  docker push ${DOCKER_REPO}/kubeletdnat-controller:${TAG}
  docker push ${DOCKER_REPO}/addons:${TAG}
  docker push ${DOCKER_REPO}/openshift-addons:${TAG}
  docker push ${DOCKER_REPO}/user-ssh-keys-agent:${TAG}
  docker push ${DOCKER_REPO}/etcd-launcher:${TAG}

  if [ "$KUBERMATIC_EDITION" == "ee" ]; then
    docker tag ${DOCKER_REPO}/api:${1} ${DOCKER_REPO}/api:${TAG}
    docker push ${DOCKER_REPO}/api:${TAG}
  fi
done
