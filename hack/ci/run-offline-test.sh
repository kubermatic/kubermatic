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

### This script was used to run KKP conformance tests inside an offline
### environment.
###
### TODO: This needs to be cleaned up greatly and adjusted to the KKP
### Operator. The presubmit job for this script is currently not used.

set -euo pipefail
# Required for signal propagation to work so
# the cleanup trap gets executed when the script
# receives a SIGINT
set -o monitor

cd "$(dirname "$0")/"
source ../lib.sh

echodate "Getting secrets from Vault"
export VAULT_ADDR=https://vault.loodse.com/
export VAULT_TOKEN=$(vault write \
  --format=json auth/approle/login \
  role_id=${VAULT_ROLE_ID} secret_id=${VAULT_SECRET_ID} \
  | jq .auth.client_token -r)

export GIT_HEAD_HASH="$(git rev-parse HEAD|tr -d '\n')"

rm -f /tmp/id_rsa
vault kv get -field=key dev/e2e-machine-controller-ssh-key > /tmp/id_rsa
vault kv get -field=pubkey dev/e2e-machine-controller-ssh-key > /tmp/id_rsa.pub
chmod 400 /tmp/id_rsa

PROXY_EXTERNAL_ADDR="$(vault kv get -field=proxy-ip dev/gcp-offline-env)"
PROXY_INTERNAL_ADDR="$(vault kv get -field=proxy-internal-ip dev/gcp-offline-env)"
KUBERNETES_CONTROLLER_ADDR="$(vault kv get -field=controller-ip dev/gcp-offline-env)"
SSH_OPTS="-i /tmp/id_rsa -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"
HELM_VERSION=$(helm version --client --template '{{.Client.SemVer}}')
BUILD_ID=${BUILD_ID:-'latest'}
NAMESPACE="prow-kubermatic-${BUILD_ID}"
VERSIONS=${VERSIONS:-'v1.13.5'}

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
  testRC=$?

  echodate "Starting cleanup"
  set +e

  # Try being a little helpful
  if [[ ${testRC} -ne 0 ]]; then
    echodate "tests failed, describing cluster"

    # Describe cluster
    kubectl describe cluster -l worker-name=${BUILD_ID}|egrep -vi 'Service Account'

    # Control plane logs
    echodate "Dumping all conntrol plane logs"
    local GOTEMPLATE='{{ range $pod := .items }}{{ range $container := .spec.containers }}{{ printf "%s,%s\n" $pod.metadata.name $container.name }}{{end}}{{end}}'
    for i in $(kubectl get pods -n $NAMESPACE -o go-template="$GOTEMPLATE"); do
      local POD="${i%,*}"
      local CONTAINER="${i#*,}"

      echo " [*] Pod $POD, container $CONTAINER:"
      kubectl logs -n "$NAMESPACE" "$POD" "$CONTAINER"
    done

    # Display machine events, we don't have to worry about secrets here as they are stored in the machine-controllers env
    # Except for vSphere
    TMP_KUBECONFIG=$(mktemp);
    USERCLUSTER_NS=$(kubectl get cluster -o name -l worker-name=${BUILD_ID} |sed 's#.kubermatic.k8s.io/#-#g')
    kubectl get secret -n ${USERCLUSTER_NS} admin-kubeconfig -o go-template='{{ index .data "kubeconfig" }}' | base64 -d > ${TMP_KUBECONFIG}
    kubectl --kubeconfig=${TMP_KUBECONFIG} describe machine -n kube-system|egrep -vi 'password|user'
  fi

  # Delete addons from all clusters that have our worker-name label
  kubectl get cluster -l worker-name=${BUILD_ID} \
     -o go-template='{{range .items}}{{.metadata.name}}{{end}}' \
     |xargs -n 1 -I ^ kubectl label addon -n cluster-^ --all worker-name-

  # Delete all clusters that have our worker-name label
  kubectl delete cluster -l worker-name=${BUILD_ID} --wait=false

  # Remove the worker-name label from all clusters that have our worker-name
  # label so the main cluster-controller will clean them up
  kubectl get cluster -l worker-name=${BUILD_ID} \
    -o go-template='{{range .items}}{{.metadata.name}}{{end}}' \
      |xargs -I ^ kubectl label cluster ^ worker-name-

  # Delete the Helm Deployment of Kubermatic
  helm delete --purge kubermatic-${BUILD_ID} --tiller-namespace=${NAMESPACE}

  # Delete the Helm installation
  kubectl delete clusterrolebinding -l prowjob=$BUILD_ID
  kubectl delete namespace ${NAMESPACE} --wait=false

  ssh -S /tmp/controller-socket -O exit root@${KUBERNETES_CONTROLLER_ADDR}
  ssh -S /tmp/proxy-socket -O exit root@${PROXY_EXTERNAL_ADDR}
  echodate "Finished cleanup"
}
trap finish EXIT

# Start docker
docker ps &>/dev/null || start-docker.sh

retry 5 docker login -u ${QUAY_IO_USERNAME} -p ${QUAY_IO_PASSWORD} quay.io

echodate "Building and pushing Docker images"

# prepare Helm charts
sed -i "s/__KUBERMATIC_TAG__/${GIT_HEAD_HASH}/g" charts/*/*.yaml
sed -i "s/__DASHBOARD_TAG__/latest/g" charts/*/*.yaml

retry 5 ./../release-docker-images.sh ${GIT_HEAD_HASH} $(git tag -l --points-at HEAD)
echodate "Successfully finished building and pushing quay images"

# Ensure we have pushed the kubermatic chart
cd ../../charts
HELM_EXTRA_ARGS="--set kubermatic.controller.image.tag=${GIT_HEAD_HASH},kubermatic.api.image.tag=${GIT_HEAD_HASH},kubermatic.masterController.image.tag=${GIT_HEAD_HASH}"
# We must not pull those images from remote. We build them on the fly
helm template ${HELM_EXTRA_ARGS} kubermatic | grep -v quay.io/kubermatic/kubermatic |../hack/retag-images.sh

# Push a tiller image
docker pull gcr.io/kubernetes-helm/tiller:${HELM_VERSION}
docker tag gcr.io/kubernetes-helm/tiller:${HELM_VERSION} 127.0.0.1:5000/kubernetes-helm/tiller:${HELM_VERSION}
docker push 127.0.0.1:5000/kubernetes-helm/tiller:${HELM_VERSION}

# Pull & Push the images for the nodes
docker pull k8s.gcr.io/pause:3.1
# Needs to match the defined image in the Seed resource's datacenter
docker tag k8s.gcr.io/pause:3.1 127.0.0.1:5000/kubernetes/pause:3.1
docker push 127.0.0.1:5000/kubernetes/pause:3.1

for VERSION in $(echo ${VERSIONS} | sed "s/,/ /g"); do
  docker pull k8s.gcr.io/hyperkube-amd64:${VERSION}
  docker tag k8s.gcr.io/hyperkube-amd64:${VERSION} 127.0.0.1:5000/kubernetes/hyperkube-amd64:${VERSION}
  docker push 127.0.0.1:5000/kubernetes/hyperkube-amd64:${VERSION}
done

# Push all kubermatic images
cd ..
KUBERMATICCOMMIT=${GIT_HEAD_HASH} GITTAG=${GIT_HEAD_HASH} make image-loader
./_build/image-loader \
  -configuration-file /dev/null \
  -addons-path addons \
  -registry 127.0.0.1:5000 \
  -log-format=Console

# Build kubermatic binaries and push the image
if ! curl -Ss --fail "http://127.0.0.1:5000/v2/kubermatic/api/tags/list"|grep -q ${GIT_HEAD_HASH}; then
  echodate "Building binaries"
  time make build
  echodate "Building docker image"
  time docker build -t 127.0.0.1:5000/kubermatic/api:${GIT_HEAD_HASH} .
  echodate "Pushing docker image"
  time retry 5 docker push 127.0.0.1:5000/kubermatic/api:${GIT_HEAD_HASH}
  echodate "Finished building and pushing docker image"
  cd -
else
  echodate "Omitting building of binaries and docker image, as tag ${GIT_HEAD_HASH} already exists in local registry"
fi

INITIAL_MANIFESTS=$(cat <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: ${NAMESPACE}
spec: {}
status: {}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
  namespace: ${NAMESPACE}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prow-${BUILD_ID}-tiller
  labels:
    prowjob: "${BUILD_ID}"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: tiller
    namespace: ${NAMESPACE}
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: prow-${BUILD_ID}-kubermatic
  labels:
    prowjob: "${BUILD_ID}"
subjects:
- kind: ServiceAccount
  name: kubermatic
  namespace: ${NAMESPACE}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
EOF
)
echodate "Creating namespace ${NAMESPACE} to deploy kubermatic in"
echo "$INITIAL_MANIFESTS"|kubectl apply -f -

echodate "Deploying tiller"
helm init \
  --tiller-image ${PROXY_INTERNAL_ADDR}:5000/kubernetes-helm/tiller:${HELM_VERSION} \
  --wait \
  --service-account=tiller \
  --tiller-namespace=${NAMESPACE}

echodate "Installing Kubermatic via Helm"
retry 3 helm upgrade --install --force --wait --timeout 300 \
  --tiller-namespace=${NAMESPACE} \
  --set=kubermatic.isMaster=true \
  --set-string=kubermatic.controller.image.tag=${GIT_HEAD_HASH} \
  --set-string=kubermatic.api.image.tag=${GIT_HEAD_HASH} \
  --set-string=kubermatic.masterController.image.tag=${GIT_HEAD_HASH} \
  --set-string=kubermatic.worker_name=${BUILD_ID} \
  --set=kubermatic.ingressClass=non-existent \
  --set=kubermatic.checks.crd.disable=true \
  --values ${VALUES_FILE} \
  --namespace ${NAMESPACE} \
  kubermatic-${BUILD_ID} charts/kubermatic/

go build --tags "$KUBERMATIC_EDITION" ./cmd/conformance-tests

cp ${KUBECONFIG} /tmp/kubeconfig-remote
kubectl --kubeconfig /tmp/kubeconfig-remote config set-cluster kubernetes --server=https://${KUBERNETES_CONTROLLER_ADDR}:6443;

SSH_OPTS="-i /tmp/id_rsa -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"
ssh ${SSH_OPTS} root@${PROXY_EXTERNAL_ADDR} "mkdir -p /tmp/${BUILD_ID}/reports"
scp ${SSH_OPTS} ./conformance-tests root@${PROXY_EXTERNAL_ADDR}:/tmp/${BUILD_ID}/conformance-tests
scp ${SSH_OPTS} /tmp/kubeconfig-remote root@${PROXY_EXTERNAL_ADDR}:/tmp/${BUILD_ID}/kubeconfig
scp ${SSH_OPTS} /tmp/id_rsa.pub root@${PROXY_EXTERNAL_ADDR}:/tmp/id_rsa.pub
ssh ${SSH_OPTS} root@${PROXY_EXTERNAL_ADDR} << EOF
  /tmp/${BUILD_ID}/conformance-tests \
    -worker-name=${BUILD_ID} \
    -kubeconfig=/tmp/${BUILD_ID}/kubeconfig \
    -node-ssh-pub-key=/tmp/id_rsa.pub \
    -kubermatic-nodes=3 \
    -kubermatic-parallel-clusters=1 \
    -name-prefix=prow-e2e \
    -reports-root=/tmp/${BUILD_ID}/reports \
    -versions="v1.13.5" \
    -providers=gcp \
    -exclude-distributions="centos,ubuntu,sles,rhel" \
    -kubermatic-delete-cluster=false \
    -only-test-creation=true \
    -gcp-service-account="${GOOGLE_SERVICE_ACCOUNT}" \
    -gcp-zone="europe-west3-a" \
    -gcp-network="global/networks/offline-network" \
    -kubevirt-kubeconfig="${KUBEVIRT_E2E_TESTS_KUBECONFIG}" \
    -gcp-subnetwork="regions/europe-west3/subnetworks/offline-subnet";
EOF
