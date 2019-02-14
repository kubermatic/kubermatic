#!/usr/bin/env bash

set -euo pipefail

export BUILD_ID=${BUILD_ID:-BUILD_ID_UNDEF}
echodate() { echo "$(date) $@"; }
echodate "Build ID is $BUILD_ID"
export VERSIONS=${VERSIONS_TO_TEST:-"v1.12.4"}
export NAMESPACE="prow-kubermatic-${BUILD_ID}"
echodate "Testing versions: ${VERSIONS}"
cd $(dirname $0)/../..
export GIT_HEAD_HASH="$(git rev-parse HEAD|tr -d '\n')"
export EXCLUDE_DISTRIBUTIONS=${EXCLUDE_DISTRIBUTIONS:-ubuntu,centos}

function retry {
  local retries=$1
  shift

  local count=0
  until "$@"; do
    exit=$?
    wait=$((2 ** $count))
    count=$(($count + 1))
    if [ $count -lt $retries ]; then
      echo "Retry $count/$retries exited $exit, retrying in $wait seconds..."
      sleep $wait
    else
      echo "Retry $count/$retries exited $exit, no more retries left."
      return $exit
    fi
  done
  return 0
}

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


if [[ ! -f $HOME/.docker/config.json ]]; then
  docker ps &>/dev/null || start-docker.sh
  mkdir  -p $HOME/.docker
  echo '{"experimental": "enabled"}' > ~/.docker/config.json
  echodate "Logging into dockerhub"
  docker login -u $DOCKERHUB_USERNAME -p $DOCKERHUB_PASSWORD
  echodate "Successfully logged into all registries"
fi

# Only build kubermatic binaries and docker image if it doesn't exist yet
# We use dockerhub because docker manifest inspect doesn't seem to work on quay
if ! docker manifest inspect docker.io/kubermatic/api:$GIT_HEAD_HASH &>/dev/null; then
  echodate "Building binaries"
  time make -C api build
  cd api
  echodate "Building docker image"
  docker build -t docker.io/kubermatic/api:${GIT_HEAD_HASH} .
  echodate "Pushing docker image"
  retry 5 docker push docker.io/kubermatic/api:${GIT_HEAD_HASH}
  echodate "Finished building and pushing docker image"
  cd -
fi

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
# --force is needed in case the first attempt at installing didn't succeed
# see https://github.com/helm/helm/pull/3597
retry 3 helm upgrade --install --force --wait --timeout 300 \
  --tiller-namespace=$NAMESPACE \
  --set=kubermatic.isMaster=true \
  --set-string=kubermatic.controller.image.repository=docker.io/kubermatic/api \
  --set-string=kubermatic.controller.image.tag=$GIT_HEAD_HASH \
  --set-string=kubermatic.api.image.repository=docker.io/kubermatic/api \
  --set-string=kubermatic.api.image.tag=$GIT_HEAD_HASH \
  --set-string=kubermatic.rbac.image.repository=docker.io/kubermatic/api \
  --set-string=kubermatic.rbac.image.tag=$GIT_HEAD_HASH \
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

echodate "Starting conformance tests"
timeout -s 9 90m ./conformance-tests \
  -debug \
  -worker-name=$BUILD_ID \
  -kubeconfig=$KUBECONFIG \
  -datacenters=$DATACENTERS_FILE \
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
  -exclude-distributions="${EXCLUDE_DISTRIBUTIONS}" \
  -kubermatic-delete-cluster=false
