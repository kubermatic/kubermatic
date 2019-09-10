#!/usr/bin/env bash
set -euo pipefail
# Required for signal propagation to work so
# the cleanup trap gets executed when the script
# receives a SIGINT
set -o monitor

cd "$(dirname "$0")/"
source ./../lib.sh

# Build and push images
./ci-push-images.sh

echodate "Getting secrets from Vault"
export VAULT_ADDR=https://vault.loodse.com/
export VAULT_TOKEN=$(vault write \
  --format=json auth/approle/login \
  role_id=${VAULT_ROLE_ID} secret_id=${VAULT_SECRET_ID} \
  | jq .auth.client_token -r)

export GIT_HEAD_HASH="$(git rev-parse HEAD|tr -d '\n')"

rm -f /tmp/id_rsa
vault kv get -field=key dev/e2e-machine-controller-ssh-key > /tmp/id_rsa
chmod 400 /tmp/id_rsa

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

ssh ${SSH_OPTS} -M -S /tmp/proxy-socket -fNT -L 5000:127.0.0.1:5000 root@${PROXY_EXTERNAL_ADDR}
ssh ${SSH_OPTS} -M -S /tmp/controller-socket -fNT -L 6443:127.0.0.1:6443 ${SSH_OPTS} \
  -o ProxyCommand="ssh ${SSH_OPTS} -W %h:%p root@${PROXY_EXTERNAL_ADDR}" \
  root@${KUBERNETES_CONTROLLER_ADDR}

# Make sure we always cleanup our sockets
function finish {
  ssh -S /tmp/controller-socket -O exit root@${KUBERNETES_CONTROLLER_ADDR}
  ssh -S /tmp/proxy-socket -O exit root@${PROXY_EXTERNAL_ADDR}
}
trap finish EXIT

# Ensure we have pushed all images from our helm chats in the local registry
cd ../../../config
helm template cert-manager | ../api/hack/retag-images.sh
helm template nginx-ingress-controller | ../api/hack/retag-images.sh
helm template oauth | ../api/hack/retag-images.sh
helm template iap | ../api/hack/retag-images.sh
helm template minio | ../api/hack/retag-images.sh
helm template s3-exporter | ../api/hack/retag-images.sh
helm template nodeport-proxy --set=nodePortProxy.image.tag=${GIT_HEAD_HASH}	\
	| ../api/hack/retag-images.sh

helm template monitoring/prometheus | ../api/hack/retag-images.sh
helm template monitoring/node-exporter | ../api/hack/retag-images.sh
helm template monitoring/kube-state-metrics | ../api/hack/retag-images.sh
helm template monitoring/grafana | ../api/hack/retag-images.sh
helm template monitoring/helm-exporter | ../api/hack/retag-images.sh
helm template monitoring/alertmanager | ../api/hack/retag-images.sh

helm template logging/elasticsearch | ../api/hack/retag-images.sh
helm template logging/fluentbit | ../api/hack/retag-images.sh
helm template logging/kibana | ../api/hack/retag-images.sh

# PULL_BASE_REF is the name of the current branch in case of a post-submit
# or the name of the base branch in case of a PR.
LATEST_DASHBOARD="$(get_latest_dashboard_hash "${PULL_BASE_REF}")"

HELM_EXTRA_ARGS="--set kubermatic.controller.image.tag=${GIT_HEAD_HASH},kubermatic.api.image.tag=${GIT_HEAD_HASH},kubermatic.masterController.image.tag=${GIT_HEAD_HASH},kubermatic.controller.addons.kubernetes.image.tag=${GIT_HEAD_HASH},kubermatic.controller.addons.openshift.image.tag=${GIT_HEAD_HASH},kubermatic.ui.image.tag=${LATEST_DASHBOARD}"
helm template ${HELM_EXTRA_ARGS} kubermatic | ../api/hack/retag-images.sh

# Push a tiller image
docker pull gcr.io/kubernetes-helm/tiller:${HELM_VERSION}
docker tag gcr.io/kubernetes-helm/tiller:${HELM_VERSION} 127.0.0.1:5000/kubernetes-helm/tiller:${HELM_VERSION}
docker push 127.0.0.1:5000/kubernetes-helm/tiller:${HELM_VERSION}

cd ../api
KUBERMATICCOMMIT=${GIT_HEAD_HASH} GITTAG=${GIT_HEAD_HASH} make image-loader
retry 6 ./_build/image-loader \
  -versions ../config/kubermatic/static/master/versions.yaml \
  -addons-path ../addons \
  -registry 127.0.0.1:5000 \
  -log-format=Console

## Deploy
HELM_INIT_ARGS="--tiller-image ${PROXY_INTERNAL_ADDR}:5000/kubernetes-helm/tiller:${HELM_VERSION}" \
  DEPLOY_NODEPORT_PROXY=false \
  DEPLOY_ALERTMANAGER=false \
  TILLER_NAMESPACE="kube-system" \
  ./hack/deploy.sh \
  master \
  ${VALUES_FILE} \
  ${HELM_EXTRA_ARGS}
