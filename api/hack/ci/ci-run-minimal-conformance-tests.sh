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
export ONLY_TEST_CREATION=${ONLY_TEST_CREATION:-false}
export PULL_BASE_REF=${PULL_BASE_REF:-$(git rev-parse --abbrev-ref HEAD)}
export PULL_BASE_SHA=${PULL_BASE_SHA:-$GIT_HEAD_HASH}
export USE_KIND=${USE_KIND:-false}

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

  fi

  # The kind scripting has its own cleanup that looks different
  if [[ $USE_KIND = "true" ]]; then
    return $testRC
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

  # Remove the custom seed so the master-controller-manager can clean it up
  # and we don't end up with a stuck Seed CR.
  kubectl delete -n $NAMESPACE seeds $SEED_NAME

  # Delete the Helm Deployment of Kubermatic
  helm delete --purge kubermatic-$BUILD_ID  \
    --tiller-namespace=$NAMESPACE

  # Delete the Helm installation
  kubectl delete clusterrolebinding -l prowjob=$BUILD_ID
  kubectl delete namespace $NAMESPACE --wait=false

  # Cleanup the endpoints objects created by the leader election
  kubectl delete endpoints -n kube-system \
    kubermatic-master-controller-manager-leader-election-$BUILD_ID kubermatic-controller-manager-$BUILD_ID

  # Upload the JUNIT files
  mv /reports/* ${ARTIFACTS}/
  echodate "Finished cleanup"
}
trap cleanup EXIT

TEST_NAME="Get Vault token"
echodate "Getting secrets from Vault"
export VAULT_ADDR=https://vault.loodse.com/
export VAULT_TOKEN=$(vault write \
  --format=json auth/approle/login \
  role_id=$VAULT_ROLE_ID secret_id=$VAULT_SECRET_ID \
  | jq .auth.client_token -r)
export KUBECONFIG=/tmp/kubeconfig
export VALUES_FILE=/tmp/values.yaml
export DATACENTERS_FILE=/tmp/datacenters.yaml
export OPENSTACK_DATACENTER_FILE=/tmp/openstack-datacenters.yaml
TEST_NAME="Get Kubeconfig from Vault"
retry 5 vault kv get -field=kubeconfig \
  dev/seed-clusters/ci.kubermatic.io > $KUBECONFIG
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
  # Build kubermatic binaries and push the image
	local current_git_hash
	current_git_hash=$(git rev-parse HEAD)
	echodate "Building containers with tag $current_git_hash"
  if ! curl -Ss --fail \
		"http://registry.registry.svc.cluster.local.:5000/v2/kubermatic/api/tags/list" \
		|grep -q "current_git_hash"; then
    mkdir -p /etc/containers
    cat <<EOF > /etc/containers/registries.conf
[registries.search]
registries = ['docker.io']
[registries.insecure]
registries = ["registry.registry.svc.cluster.local:5000"]
EOF
    echodate "Building binaries"
    TEST_NAME="Build Kubermatic binaries"
    time retry 1 make -C api build
    (
      echodate "Building docker image"
      TEST_NAME="Build Kubermatic Docker image"
      cd api
      time retry 5 buildah build-using-dockerfile --squash -t "registry.registry.svc.cluster.local:5000/kubermatic/api:$current_git_hash" .
    )
    (
      echodate "Building addons image"
      TEST_NAME="Build addons Docker image"
      cd addons
      time retry 5 buildah build-using-dockerfile --squash -t "registry.registry.svc.cluster.local:5000/kubermatic/addons:$current_git_hash" .
    )
    (
      echodate "Building openshift addons image"
      TEST_NAME="Build openshift Docker image"
      cd openshift_addons
      time retry 5 buildah build-using-dockerfile --squash -t "registry.registry.svc.cluster.local:5000/kubermatic/openshift_addons:$current_git_hash" .
    )
    (
      echodate "Building dnatcontroller image"
      TEST_NAME="Build dnatcontroller Docker image"
      cd api/cmd/kubeletdnat-controller
      make build
      time retry 5 buildah build-using-dockerfile --squash -t "registry.registry.svc.cluster.local:5000/kubermatic/kubeletdnat-controller:$current_git_hash ."
    )
    (
      echodate "Building user-ssh-keys-agent image"
      TEST_NAME="Build user-ssh-keys-agent Docker image"
      cd api/cmd/user-ssh-keys-agent
      make build
      time retry 5 buildah build-using-dockerfile --squash -t "quay.io/kubermatic/user-ssh-keys-agent:$current_git_hash ."
    )
    echodate "Pushing docker image"
    TEST_NAME="Push Kubermatic Docker image"
    time retry 5 buildah push "registry.registry.svc.cluster.local:5000/kubermatic/api:$current_git_hash"
    TEST_NAME="Push addons Docker image"
    echodate "Pushing addons image"
    time retry 5 buildah push "registry.registry.svc.cluster.local:5000/kubermatic/addons:$current_git_hash"
    TEST_NAME="Push openshift addons Docker image"
    echodate "Pushing openshift addons image"
    time retry 5 buildah push "registry.registry.svc.cluster.local:5000/kubermatic/openshift_addons:$current_git_hash"
    TEST_NAME="Push dnatcontroller Docker image"
    echodate "Pushing dnatcontroller image"
    time retry 5 buildah push "registry.registry.svc.cluster.local:5000/kubermatic/kubeletdnat-controller:$current_git_hash"
    TEST_NAME="Push user-ssh-keys-agent Docker image"
    echodate "Pushing user-ssh-keys-agent image"
    retry 5 buildah login -u "$QUAY_IO_USERNAME" -p "$QUAY_IO_PASSWORD" quay.io
    time retry 5 buildah push "quay.io/kubermatic/user-ssh-keys-agent:$current_git_hash"
    echodate "Finished building and pushing docker images"
  else
    echodate "Omitting building of binaries and docker image, as tag $current_git_hash already exists in local registry"
  fi
}

if [[ -n ${UPGRADE_TEST_BASE_HASH:-} ]]; then
  echodate "Upgradetest, checking out revision ${UPGRADE_TEST_BASE_HASH}"
  git checkout $UPGRADE_TEST_BASE_HASH
fi

build_tag_if_not_exists

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


# Hardcoded as the only thing these tests test about the dashboard is that the pod comes up. In order to
# not introduce a dependency on the dashboard push postsubmit being successfully run, we just harcode it
# here.
LATEST_DASHBOARD=43037e8f118f0e310cfcae713bc2b3bd1a2c8496

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
  --set-string=kubermatic.controller.addons.kubernetes.image.tag=${UPGRADE_TEST_BASE_HASH:-$GIT_HEAD_HASH} \
  --set-string=kubermatic.controller.addons.kubernetes.image.repository=127.0.0.1:5000/kubermatic/addons \
  --set-string=kubermatic.controller.image.tag=${UPGRADE_TEST_BASE_HASH:-$GIT_HEAD_HASH} \
  --set-string=kubermatic.controller.image.repository=127.0.0.1:5000/kubermatic/api \
  --set-string=kubermatic.controller.addons.openshift.image.tag=${GIT_HEAD_HASH} \
  --set-string=kubermatic.controller.addons.openshift.image.repository=127.0.0.1:5000/kubermatic/openshift_addons \
  --set-string=kubermatic.api.image.repository=127.0.0.1:5000/kubermatic/api \
  --set-string=kubermatic.api.image.tag=${UPGRADE_TEST_BASE_HASH:-$GIT_HEAD_HASH} \
  --set-string=kubermatic.masterController.image.tag=${UPGRADE_TEST_BASE_HASH:-$GIT_HEAD_HASH} \
  --set-string=kubermatic.masterController.image.repository=127.0.0.1:5000/kubermatic/api \
  --set-string=kubermatic.ui.image.tag=${LATEST_DASHBOARD} \
  --set-string=kubermatic.kubermaticImage=127.0.0.1:5000/kubermatic/api \
  --set-string=kubermatic.dnatcontrollerImage=127.0.0.1:5000/kubermatic/kubeletdnat-controller \
  --set-string=kubermatic.worker_name=$BUILD_ID \
  --set=kubermatic.ingressClass=non-existent \
  --set=kubermatic.checks.crd.disable=true \
  --set=kubermatic.datacenters='' \
  --set=kubermatic.dynamicDatacenters=true \
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
  kubeconfig: "$(cat $KUBECONFIG|base64 -w0)"
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
retry 7 kubectl apply -f $SEED_MANIFEST
echodate "Finished installing seed"

# We build the CLI after deploying to make sure we fail fast if the helm deployment fails
if ! ls ./api/_build/conformance-tests &>/dev/null; then
  echodate "Building conformance-tests cli"
  time make -C api conformance-tests
  echodate "Finished building conformance-tests cli"
fi

echodate "Starting conformance tests"
export KUBERMATIC_APISERVER_ADDRESS="kubermatic-api.prow-kubermatic-${BUILD_ID}.svc.cluster.local.:80"

# Gather the total time it takes between starting this sscript and staring the conformance tester
setup_elasped_time=$((${SECONDS:-} - $setup_start_time))
TEST_NAME="Setup Kubermatic total" write_junit "0" "$setup_elasped_time"

./api/hack/ci/ci-run-conformance-tester.sh
