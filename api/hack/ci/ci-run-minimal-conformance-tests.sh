#!/usr/bin/env bash

set -euo pipefail
# Required for signal propagation to work so
# the cleanup trap gets executed when the script
# receives a SIGINT
set -o monitor

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic

source ./api/hack/lib.sh

export BUILD_ID=${BUILD_ID:-BUILD_ID_UNDEF}
echodate "Build ID is $BUILD_ID"
export VERSIONS=${VERSIONS_TO_TEST:-"v1.12.4"}
export NAMESPACE="prow-kubermatic-${BUILD_ID}"
echodate "Testing versions: ${VERSIONS}"
export GIT_HEAD_HASH="$(git rev-parse HEAD|tr -d '\n')"
export EXCLUDE_DISTRIBUTIONS=${EXCLUDE_DISTRIBUTIONS:-ubuntu,centos}

function cleanup {
  testRC=$?

  echodate "Starting cleanup"
  set +e

  # Try being a little helpful
  if [[ ${testRC} -ne 0 ]]; then
    echodate "tests failed, describing cluster"
    # TODO: If this runs on something other than AWS, we need to adjust the egrep expression
    kubectl describe cluster -l worker-name=$BUILD_ID|egrep -vi 'Secret Access Key|Access Key Id'
  fi

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
  kubectl delete namespace $NAMESPACE --wait=false

  # Upload the JUNIT files
  mv /reports/* ${ARTIFACTS}/
  echodate "Finished cleanup"
}
trap cleanup EXIT

echodate "Getting secrets from Vault"
export VAULT_ADDR=https://vault.loodse.com/
export VAULT_TOKEN=$(vault write \
  --format=json auth/approle/login \
  role_id=$VAULT_ROLE_ID secret_id=$VAULT_SECRET_ID \
  | jq .auth.client_token -r)
export KUBECONFIG=/tmp/kubeconfig
export VALUES_FILE=/tmp/values.yaml
export DATACENTERS_FILE=/tmp/datacenters.yaml
vault kv get -field=kubeconfig \
  dev/seed-clusters/ci.kubermatic.io > $KUBECONFIG
vault kv get -field=values.yaml \
  dev/seed-clusters/ci.kubermatic.io > $VALUES_FILE
vault kv get -field=datacenters.yaml \
  dev/seed-clusters/ci.kubermatic.io > $DATACENTERS_FILE
echodate "Successfully got secrets from Vault"


# Build kubermatic binaries and push the image to quay
docker ps &>/dev/null || start-docker.sh
echodate "Logging into quay"
docker login -u $QUAY_IO_USERNAME -p $QUAY_IO_PASSWORD quay.io
echodate "Successfully logged into quay"

echodate "Building binaries"
time make -C api build
cd api
echodate "Building quay image"
time docker build -t quay.io/kubermatic/api:${GIT_HEAD_HASH} .
echodate "Pushing quay image"
time retry 5 docker push quay.io/kubermatic/api:${GIT_HEAD_HASH}
echodate "Finished building and pushing quay image"
cd -

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

if [[ -n ${UPGRADE_TEST_BASE_HASH:-} ]]; then
  echodate "Upgradetest, checking out revision ${UPGRADE_TEST_BASE_HASH}"
  git checkout $UPGRADE_TEST_BASE_HASH
fi
# We must delete all templates for cluster-scoped resources
# because those already exist because of the main Kubermatic installation
# otherwise the helm upgrade --install fails
rm -f config/kubermatic/templates/cluster-role-binding.yaml
rm -f config/kubermatic/templates/vpa-*
# --force is needed in case the first attempt at installing didn't succeed
# see https://github.com/helm/helm/pull/3597
retry 3 helm upgrade --install --force --wait --timeout 300 \
  --tiller-namespace=$NAMESPACE \
  --set=kubermatic.isMaster=true \
  --set-string=kubermatic.controller.image.tag=${UPGRADE_TEST_BASE_HASH:-$GIT_HEAD_HASH} \
  --set-string=kubermatic.api.image.repository=quay.io/kubermatic/api \
  --set-string=kubermatic.api.image.tag=${UPGRADE_TEST_BASE_HASH:-$GIT_HEAD_HASH} \
  --set-string=kubermatic.masterController.image.tag=${UPGRADE_TEST_BASE_HASH:-$GIT_HEAD_HASH} \
  --set-string=kubermatic.rbac.image.tag=${UPGRADE_TEST_BASE_HASH:-$GIT_HEAD_HASH} \
  --set-string=kubermatic.worker_name=$BUILD_ID \
  --set=kubermatic.ingressClass=non-existent \
  --set=kubermatic.checks.crd.disable=true \
  --values ${VALUES_FILE} \
  --namespace $NAMESPACE \
  kubermatic-$BUILD_ID ./config/kubermatic/


echodate "Finished installing Kubermatic"

# We build the CLI after deploying to make sure we fail fast if the helm deployment fails
echodate "Building conformance-tests cli"
time go build -v github.com/kubermatic/kubermatic/api/cmd/conformance-tests

echodate "Finished building conformance-tests cli"
if [[ -n ${UPGRADE_TEST_BASE_HASH:-} ]]; then
  echodate "Upgradetest, going back to old revision"
  git checkout -
fi

echodate "Starting conformance tests"
timeout -s 9 90m ./conformance-tests \
  -debug \
  -worker-name=$BUILD_ID \
  -kubeconfig=$KUBECONFIG \
  -datacenters=$DATACENTERS_FILE \
  -kubermatic-nodes=3 \
  -kubermatic-parallel-clusters=1 \
  -name-prefix=prow-e2e \
  -reports-root=/reports \
  -cleanup-on-start=false \
  -aws-access-key-id="$AWS_E2E_TESTS_KEY_ID" \
  -aws-secret-access-key="$AWS_E2E_TESTS_SECRET" \
  -versions="$VERSIONS" \
  -providers=aws \
  -exclude-distributions="${EXCLUDE_DISTRIBUTIONS}" \
  -kubermatic-delete-cluster=false

# No upgradetest, just exit
if [[ -z ${UPGRADE_TEST_BASE_HASH:-} ]]; then
  echodate "Success!"
  exit 0
fi

echodate "Installing current version of Kubermatic"
retry 3 helm upgrade --install --force --wait --timeout 300 \
  --tiller-namespace=$NAMESPACE \
  --set=kubermatic.isMaster=true \
  --set-string=kubermatic.controller.image.tag=$GIT_HEAD_HASH \
  --set-string=kubermatic.api.image.repository=quay.io/kubermatic/api \
  --set-string=kubermatic.api.image.tag=$GIT_HEAD_HASH \
  --set-string=kubermatic.masterController.image.tag=$GIT_HEAD_HASH \
  --set-string=kubermatic.rbac.image.tag=${UPGRADE_TEST_BASE_HASH:-$GIT_HEAD_HASH} \
  --set-string=kubermatic.worker_name=$BUILD_ID \
  --set=kubermatic.ingressClass=non-existent \
  --set=kubermatic.checks.crd.disable=true \
  --values ${VALUES_FILE} \
  --namespace $NAMESPACE \
  kubermatic-$BUILD_ID ./config/kubermatic/
echodate "Successfully installed current version of Kubermatic"

# We have to rebuild it so it is based on the newer Kubermatic
echodate "Building conformance-tests cli"
time go build -v github.com/kubermatic/kubermatic/api/cmd/conformance-tests

echodate "Running conformance tester with existing cluster"

# We increase the number of nodes to make sure creation
# of nodes still work
timeout -s 9 60m ./conformance-tests \
  -debug \
  -existing-cluster-label=worker-name=$BUILD_ID \
  -worker-name=$BUILD_ID \
  -kubeconfig=$KUBECONFIG \
  -datacenters=$DATACENTERS_FILE \
  -kubermatic-nodes=5 \
  -kubermatic-parallel-clusters=1 \
  -kubermatic-delete-cluster=true \
  -name-prefix=prow-e2e \
  -reports-root=/reports \
  -cleanup-on-start=false \
  -aws-access-key-id="$AWS_E2E_TESTS_KEY_ID" \
  -aws-secret-access-key="$AWS_E2E_TESTS_SECRET" \
  -versions="$VERSIONS" \
  -providers=aws \
  -exclude-distributions="${EXCLUDE_DISTRIBUTIONS}" \
  -kubermatic-delete-cluster=false
