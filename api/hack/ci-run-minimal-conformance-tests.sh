#!/usr/bin/env bash

set -euo pipefail

export BUILD_ID=${BUILD_ID:-BUILD_ID_UNDEF}
echodate() { echo "$(date) $@"; }
echodate "Build ID is $BUILD_ID"
export VERSIONS=${VERSIONS_TO_TEST:-"v1.12.4"}
export NAMESPACE="prow-kubermatic-${BUILD_ID}"
echodate "Testing versions: ${VERSIONS}"
cd $(dirname $0)/../..

function cleanup {
  echodate "Starting cleanup"
  set +e
  # Delete addons from all clusters that have our worker-name label
  kubectl get cluster -l worker-name=$BUILD_ID \
     -o go-template='{{range .items}}{{.metadata.name}}{{end}}' \
     |xargs -n 1 -I ^ kubectl label addon -n cluster-^ --all worker-name-

  # Delete all clusters that have our worker-name label
  kubectl delete cluster -l worker-name=$BUILD_ID --wait=false

  # Remove the worker-name label from all clusters that have our worker-name
  # label so the main cluster-controller will clean them up
  kubectl get cluster -l worker-name=$BUILD_ID \
    -o go-template='{{range .items}}{{.metadata.name}}{{end}}' \
      |xargs -I ^ kubectl label cluster ^ worker-name-

  # Delete the Helm Deployment of Kubermatic
  helm delete --purge kubermatic-$BUILD_ID  \
    --tiller-namespace=$NAMESPACE

  # Delete the Helm installation
  kubectl delete clusterrolebinding -l prowjob=$BUILD_ID
  kubectl delete namespace $NAMESPACE
  echodate "Finished cleanup"
}
trap cleanup EXIT SIGINT SIGTERM

docker ps &>/dev/null || start-docker.sh

echodate "Unlocking secrets repo"
cd $(go env GOPATH)/src/github.com/kubermatic/secrets
echodate $KUBERMATIC_SECRETS_GPG_KEY_BASE64 | base64 -d > /tmp/git-crypt-key
git-crypt unlock /tmp/git-crypt-key
cd -
echodate "Successfully unlocked secrets repo"

echodate "Getting secrets from Vault"
export VAULT_ADDR=https://vault.loodse.com/
export VAULT_TOKEN=$(vault write \
  --format=json auth/approle/login \
  role_id=$VAULT_ROLE_ID secret_id=$VAULT_SECRET_ID \
  | jq .auth.client_token -r)
vault kv get -field=kubeconfig \
  dev/seed-clusters/dev.kubermatic.io > /tmp/kubeconfig
vault kv get -field=values.yaml \
  dev/seed-clusters/dev.kubermatic.io > /tmp/values.yaml
export KUBECONFIG=/tmp/kubeconfig
export VALUES_FILE=/tmp/values.yaml
echodate "Successfully got secrets from Vault"


if [[ ! -f $HOME/.docker/config.json ]]; then
  echodate "Logging into quay.io"
  docker login -u $QUAY_IO_USERNAME -p $QUAY_IO_PASSWORD quay.io
  echodate "Logging into dockerhub"
  docker login -u $DOCKERHUB_USERNAME -p $DOCKERHUB_PASSWORD
  echodate "Successfully logged into all registries"
fi

echodate "Building conformance-tests cli"
time go build -v github.com/kubermatic/kubermatic/api/cmd/conformance-tests
echodate "Building kubermatic-controller-manager"
time make -C api build
echodate "Finished building conformance-tests and kubermatic-controller-manager"

echodate "Building docker image"
./api/hack/push_image.sh ${PULL_PULL_SHA}
echodate "Finished building and pushing docker images"

INITIAL_MANIFESTS=$(cat <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: $NAMESPACE
spec: {}
status: {}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
  namespace: $NAMESPACE
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
    namespace: $NAMESPACE
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
  namespace: $NAMESPACE
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
EOF
)
echodate "Creating namespace $NAMESPACE to deploy kubermatic in"
echo "$INITIAL_MANIFESTS"|kubectl apply -f -

echodate "Deploying tiller"
helm init --wait --service-account=tiller --tiller-namespace=$NAMESPACE

echodate "Installing Kubermatic via Helm"
rm -f config/kubermatic/templates/cluster-role-binding.yaml
helm upgrade --install --wait --timeout 300 \
  --tiller-namespace=$NAMESPACE \
  --set=kubermatic.isMaster=true \
  --set=kubermatic.controller.image.tag=$PULL_PULL_SHA \
  --set=kubermatic.api.image.tag=$PULL_PULL_SHA \
  --set=kubermatic.rbac.image.tag=$PULL_PULL_SHA \
  --set-string=kubermatic.worker_name=$BUILD_ID \
  --set=kubermatic.deployVPA=false \
  --set=kubermatic.ingressClass=non-existent \
  --values ${VALUES_FILE} \
  --namespace $NAMESPACE \
  kubermatic-$BUILD_ID ./config/kubermatic/
echodate "Finished installing Kubermatic"

echodate "Starting conformance tests"
./conformance-tests \
  -debug \
  -worker-name=$BUILD_ID \
  -kubeconfig=$KUBECONFIG \
  -datacenters=$(go env GOPATH)/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  -kubermatic-nodes=3 \
  -kubermatic-parallel-clusters=11 \
  -kubermatic-delete-cluster=true \
  -name-prefix=prow-e2e \
  -reports-root=/reports \
  -cleanup-on-start=false \
  -run-kubermatic-controller-manager=false \
  -aws-access-key-id="$AWS_E2E_TESTS_KEY_ID" \
  -aws-secret-access-key="$AWS_E2E_TESTS_SECRET" \
  -versions="$VERSIONS" \
  -providers=aws \
  -exclude-distributions="ubuntu,centos"
