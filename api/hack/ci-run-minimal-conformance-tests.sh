#!/usr/bin/env bash

set -euo pipefail

export BUILD_ID=${BUILD_ID:-BUILD_ID_UNDEF}
echo "Build ID is $BUILD_ID"
export VERSIONS=${VERSIONS_TO_TEST:-"v1.12.4"}
export NAMESPACE="prow-kubermatic-${BUILD_ID}"
echo "Testing versions: ${VERSIONS}"
cd $(dirname $0)/../..

function cleanup {
  echo "Starting cleanup"
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
  echo "Finished cleanup"
}
trap cleanup EXIT

docker ps &>/dev/null || start-docker.sh

echo "Unlocking secrets repo"
cd $(go env GOPATH)/src/github.com/kubermatic/secrets
echo $KUBERMATIC_SECRETS_GPG_KEY_BASE64 | base64 -d > /tmp/git-crypt-key
git-crypt unlock /tmp/git-crypt-key
cd -
echo "Successfully unlocked secrets repo"

echo "Getting secrets from Vault"
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
echo "Successfully got secrets from Vault"


if [[ ! -f $HOME/.docker/config.json ]]; then
  echo "Logging into quay.io"
  docker login -u $QUAY_IO_USERNAME -p $QUAY_IO_PASSWORD quay.io
  echo "Logging into dockerhub"
  docker login -u $DOCKERHUB_USERNAME -p $DOCKERHUB_PASSWORD
  echo "Successfully logged into all registries"
fi

echo "Building conformance-tests cli"
time go build -v github.com/kubermatic/kubermatic/api/cmd/conformance-tests
echo "Building kubermatic-controller-manager"
time make -C api kubermatic-controller-manager
echo "Finished building conformance-tests and kubermatic-controller-manager"

echo "Building docker image"
./api/hack/push_image.sh ${PULL_PULL_SHA}
echo "Finished building and pushing docker images"

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
echo "Creating namespace $NAMESPACE to deploy kubermatic in"
echo "$INITIAL_MANIFESTS"|kubectl apply -f -

echo "Deploying tiller"
helm init --wait --service-account=tiller --tiller-namespace=$NAMESPACE

echo "Installing Kubermatic via Helm"
rm -f config/kubermatic/templates/cluster-role-binding.yaml
helm upgrade --install --wait --timeout 300 \
  --tiller-namespace=$NAMESPACE \
  --set=kubermatic.isMaster=true \
  --set=kubermatic.controller.image.tag=$PULL_PULL_SHA \
  --set=kubermatic.api.image.tag=$PULL_PULL_SHA \
  --set=kubermatic.rbac.image.tag=$PULL_PULL_SHA \
  --set=kubermatic.worker_name=prow-$BUILD_ID \
  --set=kubermatic.project_migrator.dry_run=true \
  --set=kubermatic.deployVPA=false \
  --values ${VALUES_FILE} \
  --namespace $NAMESPACE \
  kubermatic-$BUILD_ID ./config/kubermatic/
echo "Finished installing Kubermatic"

echo "Starting conformance tests"
./conformance-tests \
  -debug \
  -worker-name=$BUILD_ID \
  -kubeconfig=$KUBECONFIG
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
