#!/usr/bin/env bash

set -euo pipefail
# Required for signal propagation to work so
# the cleanup trap gets executed when the script
# receives a SIGINT
set -o monitor

setup_start_time=${SECONDS:-0}

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic
export SEED_NAME=prow-build-cluster

source ./api/hack/lib.sh

export BUILD_ID=${BUILD_ID:-BUILD_ID_UNDEF}
echodate "Build ID is $BUILD_ID"
export VERSIONS=${VERSIONS_TO_TEST:-"v1.12.4"}
export NAMESPACE="prow-kubermatic-${BUILD_ID}"
echodate "Testing versions: ${VERSIONS}"
export GIT_HEAD_HASH="$(git rev-parse HEAD|tr -d '\n')"
export EXCLUDE_DISTRIBUTIONS=${EXCLUDE_DISTRIBUTIONS:-ubuntu,centos}
export DEFAULT_TIMEOUT_MINUTES=${DEFAULT_TIMEOUT_MINUTES:-10}
export ONLY_TEST_CREATION=${ONLY_TEST_CREATION:-false}
export PULL_BASE_REF=${PULL_BASE_REF:-$(git rev-parse --abbrev-ref HEAD)}
export PULL_BASE_SHA=${PULL_BASE_SHA:-$GIT_HEAD_HASH}

KUBERMATIC_PATH=$(go env GOPATH)/src/github.com/kubermatic/kubermatic
KUBERMATIC_CRD_PATH=${KUBERMATIC_PATH}/config/kubermatic/crd
KUBERMATIC_HELM_PATH=${KUBERMATIC_PATH}/helm/kubermatic
KUBERMATIC_IMAGE="quay.io/kubermatic"
KUBERMATIC_IMAGE_TAG=${1:-"latest"}

# TODO alvaroaleman: Put that into the docker image
iptables -t mangle -A POSTROUTING -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu

KUBECONFIG_PATH=~/.kube/config

SCRIPT_PATH=$(dirname -- "$(readlink -f -- "$BASH_SOURCE")")

DEX_PATH=${SCRIPT_PATH}/helm/oauth
DEX_NAMESPACE="oauth"

# if no provider argument has been specified, default to aws
provider=${PROVIDER:-"aws"}

if [[ -n ${OPENSHIFT:-} ]]; then
  OPENSHIFT_ARG="-openshift=true"
  export VERSIONS=${OPENSHIFT_VERSION}
  OPENSHIFT_HELM_ARGS="--set-string=kubermatic.controller.featureGates=OpenIDAuthPlugin=true
 --set-string=kubermatic.auth.caBundle=$(cat /etc/oidc-data/oidc-ca-file|base64 -w0)
 --set-string=kubermatic.auth.tokenIssuer=$OIDC_ISSUER_URL
 --set-string=kubermatic.auth.issuerClientID=$OIDC_ISSUER_CLIENT_ID
 --set-string=kubermatic.auth.issuerClientSecret=$OIDC_ISSUER_CLIENT_SECRET"
fi

function cleanup {
  testRC=$?
	sleep 1h

  echodate "Starting cleanup"
  set +e

  # Try being a little helpful
  if [[ ${testRC} -ne 0 ]]; then
    echodate "tests failed, describing cluster"

    # Describe cluster
    if [[ $provider == "aws" ]]; then
      kubectl describe cluster -l worker-name=$BUILD_ID|egrep -vi 'Secret Access Key|Access Key Id'
    elif [[ $provider == "packet" ]]; then
      kubectl describe cluster -l worker-name=$BUILD_ID|egrep -vi 'APIKey|ProjectID'
    elif [[ $provider == "gcp" ]]; then
      kubectl describe cluster -l worker-name=$BUILD_ID|egrep -vi 'Service Account'
    elif [[ $provider == "azure" ]]; then
      kubectl describe cluster -l worker-name=$BUILD_ID|egrep -vi 'ClientID|ClientSecret|SubscriptionID|TenantID'
    elif [[ $provider == "digitalocean" ]]; then
      kubectl describe cluster -l worker-name=$BUILD_ID|egrep -vi 'Token'
    elif [[ $provider == "hetzner" ]]; then
      kubectl describe cluster -l worker-name=$BUILD_ID|egrep -vi 'Token'
    elif [[ $provider == "openstack" ]]; then
      kubectl describe cluster -l worker-name=$BUILD_ID|egrep -vi 'Domain|Tenant|Username|Password'
    elif [[ $provider == "vsphere" ]]; then
      kubectl describe cluster -l worker-name=$BUILD_ID|egrep -vi 'Username|Password'
    elif [[ $provider == "kubevirt" ]]; then
      kubectl describe cluster -l worker-name=$BUILD_ID|grep Events: -A 100
    else
      echo "Provider $provider is not yet supported."
      exit 1
    fi

    # Control plane logs
    echodate "Dumping all conntrol plane logs"
    local GOTEMPLATE='{{ range $pod := .items }}{{ range $container := .spec.containers }}{{ printf "%s,%s\n" $pod.metadata.name $container.name }}{{end}}{{end}}'
    for i in $(kubectl get pods -n $NAMESPACE -o go-template="$GOTEMPLATE"); do
      local POD="${i%,*}"
      local CONTAINER="${i#*,}"

      echo " [*] Pod $POD, container $CONTAINER:"
      kubectl logs -n $NAMESPACE "$POD" "$CONTAINER"
    done

    # Display machine events, we don't have to worry about secrets here as they are stored in the machine-controllers env
    # Except for vSphere
    TMP_KUBECONFIG=$(mktemp);
    USERCLUSTER_NS=$(kubectl get cluster -o name -l worker-name=${BUILD_ID} |sed 's#.kubermatic.k8s.io/#-#g')
    kubectl get secret -n ${USERCLUSTER_NS} admin-kubeconfig -o go-template='{{ index .data "kubeconfig" }}' | base64 -d > $TMP_KUBECONFIG
    kubectl --kubeconfig=${TMP_KUBECONFIG} describe machine -n kube-system|egrep -vi 'password|user'
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

  # Delete kind cluster
  kind delete cluster --name ${SEED_NAME}

  # Upload the JUNIT files
  mv /reports/* ${ARTIFACTS}/
  echodate "Finished cleanup"
}
trap cleanup EXIT

# Create kind cluster
TEST_NAME="Create kind cluster"
echodate "Creating the kind cluster"
kind create cluster --name ${SEED_NAME}
cp ~/.kube/kind-config-${SEED_NAME} ~/.kube/config
export KUBECONFIG="$(kind get kubeconfig-path --name=${SEED_NAME})"

# Prepare the kind cluster
TEST_NAME="Preparing the kind cluster"
echodate "Preparing the kind cluster"
## Delete kind default storage class
kubectl delete storageclass standard

## Create Kubermatic CRDs
kubectl apply -f ${KUBERMATIC_CRD_PATH}

TEST_NAME="Get kind cluster Kubeconfig"
echodate "Get kind cluster Kubeconfig"
KUBECONFIG_PARSED=$(mktemp)
cp ${KUBECONFIG} ${KUBECONFIG_PARSED}
KUBECONFIG_PATH=${KUBECONFIG_PARSED}
yq w -i ${KUBECONFIG_PATH} clusters[0].name ${SEED_NAME}
yq w -i ${KUBECONFIG_PATH} contexts[0].name ${SEED_NAME}
yq w -i ${KUBECONFIG_PATH} contexts[0].context.cluster ${SEED_NAME}
yq w -i ${KUBECONFIG_PATH} contexts[0].context.user ${SEED_NAME}
yq w -i ${KUBECONFIG_PATH} current-context ${SEED_NAME}
yq w -i ${KUBECONFIG_PATH} users[0].name ${SEED_NAME}
yq w -i ${KUBECONFIG_PATH} clusters[0].cluster.server https://kubernetes.default:443
KUBECONFIG_ENCODED=$(base64 ${KUBECONFIG_PATH} | tr -d '\n')

TEST_NAME="Get Vault token"
echodate "Getting secrets from Vault"
export VAULT_ADDR=https://vault.loodse.com/
export VAULT_TOKEN=$(vault write \
  --format=json auth/approle/login \
  role_id=$VAULT_ROLE_ID secret_id=$VAULT_SECRET_ID \
  | jq .auth.client_token -r)
export VALUES_FILE=/tmp/values.yaml
export DATACENTERS_FILE=/tmp/datacenters.yaml
export OPENSTACK_DATACENTER_FILE=/tmp/openstack-datacenters.yaml

TEST_NAME="Get Values file from Vault"
retry 5 vault kv get -field=values.yaml \
  dev/seed-clusters/ci.kubermatic.io > $VALUES_FILE
TEST_NAME="Get datacenters.yaml from Vault"
retry 5 vault kv get -field=datacenters.yaml \
  dev/seed-clusters/ci.kubermatic.io > $DATACENTERS_FILE
TEST_NAME="Get Openstack datacenters file from Vault"
retry 5 vault kv get -field=openstack-datacenter.yaml \
  dev/seed-clusters/ci.kubermatic.io > $OPENSTACK_DATACENTER_FILE
TEST_NAME="Get ProjectID from Vault"
retry 5 vault kv get -field=project_id \
  dev/seed-clusters/ci.kubermatic.io > /tmp/kubermatic_project_id
export KUBERMATIC_PROJECT_ID="$(cat /tmp/kubermatic_project_id)"
TEST_NAME="Get ServiceAccount token from Vault"
retry 5 vault kv get -field=serviceaccount_token \
  dev/seed-clusters/ci.kubermatic.io > /tmp/kubermatic_serviceaccount_token
export KUBERMATIC_SERVICEACCOUNT_TOKEN="$(cat /tmp/kubermatic_serviceaccount_token)"
echodate "Successfully got secrets from Vault"

build_tag_if_not_exists() {
    echodate "Building binaries"
    TEST_NAME="Build Kubermatic binaries"
    time retry 1 make -C api build
    (
      echodate "Building docker image"
      TEST_NAME="Build Kubermatic Docker image"
      cd api
      docker build -t ${KUBERMATIC_IMAGE}/api:${KUBERMATIC_IMAGE_TAG} .
    )
    (
      echodate "Building addons image"
      TEST_NAME="Build addons Docker image"
      cd addons
      docker build -t ${KUBERMATIC_IMAGE}/addons:${KUBERMATIC_IMAGE_TAG} .
    )
    (
      echodate "Building openshift addons image"
      TEST_NAME="Build openshift Docker image"
      cd openshift_addons
      docker build -t ${KUBERMATIC_IMAGE}/openshift-addons:${KUBERMATIC_IMAGE_TAG} .
    )
    (
      echodate "Building dnatcontroller image"
      TEST_NAME="Build dnatcontroller Docker image"
      cd api/cmd/kubeletdnat-controller
      make build
      docker build -t ${KUBERMATIC_IMAGE}/kubeletdnat-controller:${KUBERMATIC_IMAGE_TAG} .
    )
    echodate "Pushing images to the kind cluster"
    TEST_NAME="Pushing images to the kind cluster"
    time retry 5 kind load docker-image ${KUBERMATIC_IMAGE}/api:${KUBERMATIC_IMAGE_TAG} --name ${SEED_NAME}
    time retry 5 kind load docker-image ${KUBERMATIC_IMAGE}/addons:${KUBERMATIC_IMAGE_TAG} --name ${SEED_NAME}
    time retry 5 kind load docker-image ${KUBERMATIC_IMAGE}/openshift-addons:${KUBERMATIC_IMAGE_TAG} --name ${SEED_NAME}
    time retry 5 kind load docker-image ${KUBERMATIC_IMAGE}/kubeletdnat-controller:${KUBERMATIC_IMAGE_TAG} --name ${SEED_NAME}
}

build_tag_if_not_exists "$GIT_HEAD_HASH"

INITIAL_MANIFESTS="$(mktemp)"
cat <<EOF >$INITIAL_MANIFESTS
apiVersion: v1
kind: Namespace
metadata:
  name: $NAMESPACE
  labels:
    worker-name: "$BUILD_ID"
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
---
EOF
echodate "Creating namespace $NAMESPACE to deploy kubermatic in"
TEST_NAME="Create Kubermatic namespace and Bindings"
retry 5 kubectl apply -f $INITIAL_MANIFESTS

echodate "Deploying tiller"
TEST_NAME="Deploy Tiller"
helm init --wait --service-account=tiller --tiller-namespace=$NAMESPACE

echodate "Installing Kubermatic via Helm"
TEST_NAME="Deploy Kubermatic"

if [[ -n ${UPGRADE_TEST_BASE_HASH:-} ]]; then
  echodate "Upgradetest, checking out revision ${UPGRADE_TEST_BASE_HASH}"
  git checkout $UPGRADE_TEST_BASE_HASH
  build_tag_if_not_exists "$UPGRADE_TEST_BASE_HASH"
fi

# Hardcoded as the only thing these tests test about the dashboard is that the pod comes up. In order to
# not introduce a dependency on the dashboard push postsubmit being successfully run, we just harcode it
# here.
LATEST_DASHBOARD=43037e8f118f0e310cfcae713bc2b3bd1a2c8496

# TODO(xmudrii): Check do we need the hostpath-provisioner.
TEST_NAME="Deploying hostpath-provisioner"
echodate "Deploying hostpath-provisioner"
LOCAL_PROVISIONER_NAMESPACE="local-provisioner"
helm repo add rimusz https://charts.rimusz.net
helm repo update
helm install --wait --timeout 180 \
	--tiller-namespace=${NAMESPACE} \
	--namespace ${LOCAL_PROVISIONER_NAMESPACE} \
	--set storageClass.name=kubermatic-fast \
	--name hostpath-provisioner rimusz/hostpath-provisioner

TEST_NAME="Deploying dex"
echodate "Deploying dex"
mkdir -p ${SCRIPT_PATH}/helm
cp -r ${KUBERMATIC_HELM_PATH}/config/oauth ${DEX_PATH}
rm ${DEX_PATH}/templates/ingress.yaml
cp ${SCRIPT_PATH}/testdata/oauth_values.yaml ${DEX_PATH}/values.yaml
cp ${SCRIPT_PATH}/testdata/oauth_configmap.yaml ${DEX_PATH}/templates/configmap.yaml
cp -r ${KUBERMATIC_HELM_PATH}/config/kubermatic ${KUBERMATIC_HELM_PATH}
rm ${KUBERMATIC_HELM_PATH}/templates/ingress.yaml

helm install --wait --timeout 180 \
  --tiller-namespace=${NAMESPACE} \
  --set-string=dex.ingress.host=http://dex.oauth:5556 \
  --values ${DEX_PATH}/values.yaml \
  --namespace ${DEX_NAMESPACE} \
  --name kubermatic-oauth-e2e ${DEX_PATH}

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
  --set=kubermatic.imagePullSecretData=$IMAGE_PULL_SECRET_DATA \
  --set=kubermatic.auth.serviceAccountKey=$SERVICE_ACCOUNT_KEY \
  --set=kubermatic.auth.tokenIssuer=http://dex.oauth:5556 \
  --set=kubermatic.auth.clientID=kubermatic \
  --set=kubermatic.kubeconfig=${KUBECONFIG_ENCODED} \
  --set=kubermatic.controller.replicas=1 \
  --set=kubermatic.controller.datacenterName=${SEED_NAME} \
  --set=kubermatic.controller.image.tag=${KUBERMATIC_IMAGE_TAG} \
  --set=kubermatic.controller.addons.kubernetes.image.tag=${KUBERMATIC_IMAGE_TAG} \
  --set=kubermatic.api.image.tag=${KUBERMATIC_IMAGE_TAG} \
  --set=kubermatic.api.replicas=1 \
  --set=kubermatic.apiserverDefaultReplicas=1 \
  --set=kubermatic.masterController.image.tag=${KUBERMATIC_IMAGE_TAG} \
  --set-string=kubermatic.ui.image.tag=${LATEST_DASHBOARD} \
  --set-string=kubermatic.worker_name=$BUILD_ID \
  --set=kubermatic.ingressClass=non-existent \
  --set=kubermatic.checks.crd.disable=true \
  --set=kubermatic.dynamicDatacenters=true \
  --set=kubermatic.deployVPA=false \
  ${OPENSHIFT_HELM_ARGS:-} \
  --values ${VALUES_FILE} \
  --namespace $NAMESPACE \
  kubermatic-$BUILD_ID ./config/kubermatic/
echodate "Finished installing Kubermatic"

echodate "Installing seed"
SEED_MANIFEST="$(mktemp)"
cat <<EOF >$SEED_MANIFEST
kind: Secret
apiVersion: v1
metadata:
  name: ${SEED_NAME}-kubeconfig
  namespace: ${NAMESPACE}
data:
  kubeconfig: "${KUBECONFIG_ENCODED}"
---
kind: Seed
apiVersion: kubermatic.k8s.io/v1
metadata:
  name: ${SEED_NAME}
  namespace: ${NAMESPACE}
  labels:
    worker-name: "$BUILD_ID"
spec:
  country: Germany
  location: Hamburg
  kubeconfig:
    name: ${SEED_NAME}-kubeconfig
    namespace: ${NAMESPACE}
    fieldPath: kubeconfig
  datacenters:
    aws-eu-central-1a:
      location: EU (Frankfurt)
      country: DE
      spec:
        aws:
          region: eu-central-1
    hetzner-nbg1:
      location: Nuremberg 1 DC 3
      country: DE
      spec:
        hetzner:
          datacenter: nbg1-dc3
    vsphere-ger:
      location: Hamburg
      country: DE
      spec:
        vsphere:
          endpoint: "https://vcenter.loodse.io"
          datacenter: "dc-1"
          datastore: "exsi-nas"
          cluster: "cl-1"
          root_path: "/dc-1/vm/e2e-tests"
          templates:
            ubuntu: "machine-controller-e2e-ubuntu"
            centos: "machine-controller-e2e-centos"
            coreos: "machine-controller-e2e-coreos"
    azure-westeurope:
      location: "Azure West europe"
      country: NL
      spec:
        azure:
          location: "westeurope"
    gcp-westeurope:
      location: "Europe West (Germany)"
      country: DE
      spec:
        gcp:
          region: europe-west3
          zone_suffixes:
          - c
    packet-ewr1:
      location: "Packet EWR1 (New York)"
      country: US
      spec:
        packet:
          facilities:
          - ewr1
    do-ams3:
      location: Amsterdam
      country: NL
      spec:
        digitalocean:
          region: ams3
    kubevirt-europe-west3-c:
      location: Frankfurt
      country: DE
      spec:
        kubevirt: {}
$(cat $OPENSTACK_DATACENTER_FILE)
EOF
TEST_NAME="Deploy Seed Manifest"
retry 5 kubectl apply -f $SEED_MANIFEST
echodate "Finished installing seed"

# We build the CLI after deploying to make sure we fail fast if the helm deployment fails
if ! ls ./api/_build/conformance-tests &>/dev/null; then
  echodate "Building conformance-tests cli"
  time make -C api conformance-tests
  echodate "Finished building conformance-tests cli"
fi

if [[ -n ${UPGRADE_TEST_BASE_HASH:-} ]]; then
  echodate "Upgradetest, going back to old revision"
  git checkout -
fi

echodate "Starting conformance tests"
export KUBERMATIC_APISERVER_ADDRESS="kubermatic-api.prow-kubermatic-${BUILD_ID}.svc.cluster.local.:80"
if [[ $provider == "aws" ]]; then
  EXTRA_ARGS="-aws-access-key-id=${AWS_E2E_TESTS_KEY_ID}
     -aws-secret-access-key=${AWS_E2E_TESTS_SECRET}"
elif [[ $provider == "packet" ]]; then
  EXTRA_ARGS="-packet-api-key=${PACKET_API_KEY}
     -packet-project-id=${PACKET_PROJECT_ID}"
elif [[ $provider == "gcp" ]]; then
  EXTRA_ARGS="-gcp-service-account=${GOOGLE_SERVICE_ACCOUNT}"
elif [[ $provider == "azure" ]]; then
  EXTRA_ARGS="-azure-client-id=${AZURE_E2E_TESTS_CLIENT_ID}
    -azure-client-secret=${AZURE_E2E_TESTS_CLIENT_SECRET}
    -azure-tenant-id=${AZURE_E2E_TESTS_TENANT_ID}
    -azure-subscription-id=${AZURE_E2E_TESTS_SUBSCRIPTION_ID}"
elif [[ $provider == "digitalocean" ]]; then
  EXTRA_ARGS="-digitalocean-token=${DO_E2E_TESTS_TOKEN}"
elif [[ $provider == "hetzner" ]]; then
  EXTRA_ARGS="-hetzner-token=${HZ_E2E_TOKEN}"
elif [[ $provider == "openstack" ]]; then
  EXTRA_ARGS="-openstack-domain=${OS_DOMAIN}
    -openstack-tenant=${OS_TENANT_NAME}
    -openstack-username=${OS_USERNAME}
    -openstack-password=${OS_PASSWORD}"
elif [[ $provider == "vsphere" ]]; then
  EXTRA_ARGS="-vsphere-username=${VSPHERE_E2E_USERNAME}
    -vsphere-password=${VSPHERE_E2E_PASSWORD}"
elif [[ $provider == "kubevirt" ]]; then
  EXTRA_ARGS="-kubevirt-kubeconfig=${KUBEVIRT_E2E_TESTS_KUBECONFIG}"
fi

# Gather the total time it takes between starting this sscript and staring the conformance tester
setup_elasped_time=$((${SECONDS:-} - $setup_start_time))
TEST_NAME="Setup Kubermatic total" write_junit "0" "$setup_elasped_time"

kubermatic_delete_cluster="true"
if [ -n "${UPGRADE_TEST_BASE_HASH:-}" ]; then
  kubermatic_delete_cluster="false"
fi

timeout -s 9 90m ./api/_build/conformance-tests ${EXTRA_ARGS:-} \
  -debug \
  -worker-name=$BUILD_ID \
  -kubeconfig=$KUBECONFIG \
  -datacenters=$DATACENTERS_FILE \
  -kubermatic-nodes=3 \
  -kubermatic-parallel-clusters=1 \
  -name-prefix=prow-e2e \
  -reports-root=/reports \
  -cleanup-on-start=false \
  -run-kubermatic-controller-manager=false \
  -versions="$VERSIONS" \
  -providers=$provider \
  -only-test-creation="${ONLY_TEST_CREATION}" \
  -exclude-distributions="${EXCLUDE_DISTRIBUTIONS}" \
  ${OPENSHIFT_ARG:-} \
  -kubermatic-delete-cluster=${kubermatic_delete_cluster} \
  -print-ginkgo-logs=true \
  -default-timeout-minutes=${DEFAULT_TIMEOUT_MINUTES} \
  -create-oidc-token=true

# TODO(xmudrii): Back to this part.
echodate "Success!"
exit 0

# # No upgradetest, just exit
# if [[ -z ${UPGRADE_TEST_BASE_HASH:-} ]]; then
#   echodate "Success!"
#   exit 0
# fi

# # PULL_BASE_REF is the name of the current branch in case of a post-submit
# # or the name of the base branch in case of a PR.
# LATEST_DASHBOARD="$(get_latest_dashboard_hash "${PULL_BASE_REF}")"

# echodate "Installing current version of Kubermatic"
# retry 3 helm upgrade --install --force --wait --timeout 300 \
#   --tiller-namespace=$NAMESPACE \
#   --set=kubermatic.isMaster=true \
#   --set-string=kubermatic.controller.addons.kubernetes.image.tag=${GIT_HEAD_HASH} \
#   --set-string=kubermatic.controller.addons.kubernetes.image.repository=127.0.0.1:5000/kubermatic/addons \
#   --set-string=kubermatic.controller.addons.openshift.image.tag=${GIT_HEAD_HASH} \
#   --set-string=kubermatic.controller.addons.openshift.image.repository=127.0.0.1:5000/kubermatic/openshift_addons \
#   --set-string=kubermatic.controller.image.tag=${GIT_HEAD_HASH} \
#   --set-string=kubermatic.controller.image.repository=127.0.0.1:5000/kubermatic/api \
#   --set-string=kubermatic.api.image.repository=127.0.0.1:5000/kubermatic/api \
#   --set-string=kubermatic.api.image.tag=${GIT_HEAD_HASH} \
#   --set-string=kubermatic.masterController.image.tag=${GIT_HEAD_HASH} \
#   --set-string=kubermatic.masterController.image.repository=127.0.0.1:5000/kubermatic/api \
#   --set-string=kubermatic.ui.image.tag=${LATEST_DASHBOARD} \
#   --set-string=kubermatic.kubermaticImage=127.0.0.1:5000/kubermatic/api \
#   --set-string=kubermatic.dnatcontrollerImage=127.0.0.1:5000/kubermatic/kubeletdnat-controller \
#   --set-string=kubermatic.worker_name=$BUILD_ID \
#   --set=kubermatic.ingressClass=non-existent \
#   --set=kubermatic.checks.crd.disable=true \
#   ${OPENSHIFT_HELM_ARGS:-} \
#   --values ${VALUES_FILE} \
#   --namespace $NAMESPACE \
#   kubermatic-$BUILD_ID ./config/kubermatic/
# echodate "Successfully installed current version of Kubermatic"

# # We have to rebuild it so it is based on the newer Kubermatic
# echodate "Building conformance-tests cli"
# time go build -v github.com/kubermatic/kubermatic/api/cmd/conformance-tests

# echodate "Running conformance tester with existing cluster"

# # We increase the number of nodes to make sure creation
# # of nodes still work
# timeout -s 9 60m ./conformance-tests $EXTRA_ARGS \
#   -debug \
#   -existing-cluster-label=worker-name=$BUILD_ID \
#   -worker-name=$BUILD_ID \
#   -kubeconfig=$KUBECONFIG \
#   -datacenters=$DATACENTERS_FILE \
#   -kubermatic-nodes=5 \
#   -kubermatic-parallel-clusters=1 \
#   -kubermatic-delete-cluster=true \
#   -name-prefix=prow-e2e \
#   -reports-root=/reports \
#   -cleanup-on-start=false \
#   -versions="$VERSIONS" \
#   -providers=$provider \
#   -only-test-creation="${ONLY_TEST_CREATION}" \
#   -exclude-distributions="${EXCLUDE_DISTRIBUTIONS}" \
#   ${OPENSHIFT_ARG:-} \
#   -print-ginkgo-logs=true \
#   -default-timeout-minutes=${DEFAULT_TIMEOUT_MINUTES}
