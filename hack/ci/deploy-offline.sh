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

### This script is used as a postsubmit job and updates the offline
### test cluster.

set -euo pipefail
# Required for signal propagation to work so
# the cleanup trap gets executed when the script
# receives a SIGINT
set -o monitor

cd "$(dirname "$0")/../.."
source hack/lib.sh

# Build and push images
./hack/ci/push-images.sh

echodate "Getting secrets from Vault"
export VAULT_TOKEN=$(vault write \
  --format=json auth/approle/login \
  role_id=${VAULT_ROLE_ID} secret_id=${VAULT_SECRET_ID} |
  jq .auth.client_token -r)

export GIT_HEAD_HASH="$(git rev-parse HEAD | tr -d '\n')"

rm -f /tmp/id_rsa
vault kv get -field=key dev/e2e-machine-controller-ssh-key > /tmp/id_rsa
chmod 400 /tmp/id_rsa

REGISTRY=127.0.0.1:5000
PROXY_EXTERNAL_ADDR="$(vault kv get -field=proxy-ip dev/gcp-offline-env)"
PROXY_INTERNAL_ADDR="$(vault kv get -field=proxy-internal-ip dev/gcp-offline-env)"
KUBERNETES_CONTROLLER_ADDR="$(vault kv get -field=controller-ip dev/gcp-offline-env)"
SSH_OPTS="-i /tmp/id_rsa -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"
HELM_VERSION=$(helm version --client --template '{{.Client.SemVer}}')
VALUES_FILE="/tmp/values.yaml"
vault kv get -field=values.yaml dev/seed-clusters/offline.kubermatic.io > ${VALUES_FILE}

vault kv get -field=kubeconfig dev/gcp-offline-env > /tmp/kubeconfig
export KUBECONFIG="/tmp/kubeconfig"
kubectl config set clusters.kubernetes.server https://127.0.0.1:6443

# port-forward the Docker registry and Kubernetes API
ssh ${SSH_OPTS} -M -S /tmp/proxy-socket -fNT -L 5000:${REGISTRY} root@${PROXY_EXTERNAL_ADDR}
ssh ${SSH_OPTS} -M -S /tmp/controller-socket -fNT -L 6443:127.0.0.1:6443 ${SSH_OPTS} \
  -o ProxyCommand="ssh ${SSH_OPTS} -W %h:%p root@${PROXY_EXTERNAL_ADDR}" \
  root@${KUBERNETES_CONTROLLER_ADDR}

# Make sure we always cleanup our sockets
function finish {
  ssh -S /tmp/controller-socket -O exit root@${KUBERNETES_CONTROLLER_ADDR}
  ssh -S /tmp/proxy-socket -O exit root@${PROXY_EXTERNAL_ADDR}
}
trap finish EXIT

# dashboard's primary branch was renamed before KKP's, so until KKP follows suite,
# we have to temporarily adjust the branch name here
UIBRANCH="${PULL_BASE_REF:-master}"
if [ "$UIBRANCH" == "master" ]; then
  UIBRANCH=main
fi

# PULL_BASE_REF is the name of the current branch in case of a post-submit
# or the name of the base branch in case of a PR.
export UIDOCKERTAG="$(get_latest_dashboard_hash "${UIBRANCH}")"
export KUBERMATICCOMMIT="${GIT_HEAD_HASH}"

make kubermatic-installer

# push all images from KKP and Helm charts to the local registry;
# do not use the VALUES_FILE for the loading process, as it already
# overrides all the Docker images and for the image-loading we want
# to get the _original_ image names.
LOADER_VALUES_FILE=/tmp/loader-values.yaml
cat << EOF > ${LOADER_VALUES_FILE}
kubermaticOperator:
  image:
    tag: ${GIT_HEAD_HASH}
  imagePullSecret: '{}'

nodePortProxy:
  image:
    tag: ${GIT_HEAD_HASH}

iap:
  deployments:
    dummy:
      name: dummy
      client_id: dummy
      client_secret: xxx
      encryption_key: xxx
      upstream_service: example.com.svc.cluster.local
      upstream_port: 9093
      ingress:
        host: "dummy.example.com"
        annotations: {}
EOF

_build/kubermatic-installer mirror-images \
  --configuration-file /dev/null \
  --addons-path addons \
  --charts-path charts \
  --helm-values-file "${LOADER_VALUES_FILE}" \
  --registry "${REGISTRY}"

# Push a tiller image
docker pull gcr.io/kubernetes-helm/tiller:${HELM_VERSION}
docker tag gcr.io/kubernetes-helm/tiller:${HELM_VERSION} ${REGISTRY}/kubernetes-helm/tiller:${HELM_VERSION}
docker push ${REGISTRY}/kubernetes-helm/tiller:${HELM_VERSION}

# Deploy
DEPLOY_STACK=kubermatic \
  ./hack/ci/deploy.sh \
  master \
  ${VALUES_FILE}

DEPLOY_STACK=monitoring \
  ./hack/ci/deploy.sh \
  master \
  ${VALUES_FILE}

DEPLOY_STACK=logging \
  ./hack/ci/deploy.sh \
  master \
  ${VALUES_FILE}
