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

cd "$(dirname "$0")/../"
source ./hack/lib.sh

REPOSUFFIX=""
KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ee}"

if [ "$KUBERMATIC_EDITION" != "ce" ]; then
  REPOSUFFIX="-$KUBERMATIC_EDITION"
fi

make build
docker build -t quay.io/kubermatic/kubermatic${REPOSUFFIX}:${1} .
cd cmd/nodeport-proxy && export TAG=${1} && make docker && unset TAG && cd -
cd cmd/kubeletdnat-controller && export TAG=${1} && make docker && unset TAG && cd -
docker build -t "quay.io/kubermatic/addons:${1}" ../addons
docker build -t "quay.io/kubermatic/openshift-addons:${1}" ../openshift_addons
cd cmd/user-ssh-keys-agent && export TAG=${1} && make docker && unset TAG && cd -

# keep a mirror of the EE version in the old repo
if [ "$KUBERMATIC_EDITION" == "ee" ]; then
  docker tag quay.io/kubermatic/kubermatic${REPOSUFFIX}:${1} quay.io/kubermatic/api:${1}
fi

for TAG in "$@"; do
  if [[ -z "$TAG" ]]; then
    continue
  fi

  echodate "Tagging ${TAG}"
  docker tag quay.io/kubermatic/kubermatic${REPOSUFFIX}:${1} quay.io/kubermatic/kubermatic${REPOSUFFIX}:${TAG}
  docker tag quay.io/kubermatic/nodeport-proxy:${1} quay.io/kubermatic/nodeport-proxy:${TAG}
  docker tag quay.io/kubermatic/kubeletdnat-controller:${1}  quay.io/kubermatic/kubeletdnat-controller:${TAG}
  docker tag quay.io/kubermatic/addons:${1} quay.io/kubermatic/addons:${TAG}
  docker tag quay.io/kubermatic/openshift-addons:${1} quay.io/kubermatic/openshift-addons:${TAG}
  docker tag quay.io/kubermatic/user-ssh-keys-agent:${1} quay.io/kubermatic/user-ssh-keys-agent:${TAG}

  docker push quay.io/kubermatic/kubermatic${REPOSUFFIX}:${TAG}
  docker push quay.io/kubermatic/nodeport-proxy:${TAG}
  docker push quay.io/kubermatic/kubeletdnat-controller:${TAG}
  docker push quay.io/kubermatic/addons:${TAG}
  docker push quay.io/kubermatic/openshift-addons:${TAG}
  docker push quay.io/kubermatic/user-ssh-keys-agent:${TAG}

  if [ "$KUBERMATIC_EDITION" == "ee" ]; then
    docker tag quay.io/kubermatic/api:${1} quay.io/kubermatic/api:${TAG}
    docker push quay.io/kubermatic/api:${TAG}
  fi
done
