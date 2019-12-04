#!/usr/bin/env bash

set -euo pipefail

#############################################################
## CI Setup Kubermatic in kind                              #
## A simple script to get a Kubermatic setup using kind     #
#############################################################
## Caller _must_ add the following block:
# function cleanup {
# 	kubectl delete service -l "prow.k8s.io/id=$PROW_JOB_ID"
#
# 	# Kill all descendant processes
# 	pkill -P $$
#
#   # Clean up clusters
#   kubectl delete cluster --all
# }
# trap cleanup EXIT

# The kubemaric version to build
export KUBERMATIC_VERSION=$(git rev-parse HEAD)
CONTROLLER_IMAGE="quay.io/kubermatic/cluster-exposer:v1.0.0"
export SEED_NAME=prow-build-cluster

if [[ -z ${JOB_NAME} ]]; then
	echo "This script should only be running in a CI environment."
	exit 1
fi

if [[ -z ${PROW_JOB_ID} ]]; then
	echo "Build id env variable has to be set."
	exit 1
fi


# Set docker config
echo $IMAGE_PULL_SECRET_DATA | base64 -d > /config.json

# Start docker daemon
dockerd > /dev/null 2> /dev/null &

# Wait for it to start
while (! docker stats --no-stream ); do
  # Docker takes a few seconds to initialize
  echo "Waiting for Docker..."
  sleep 1
done

# Load kind image
docker load --input /kindest.tar

# Prevent mtu-related timeouts
iptables -t mangle -A POSTROUTING -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu

# Make debugging a bit better
cat <<EOF >>~/.bashrc
cn ()
{
    kubectl config set-context \$(kubectl config current-context) --namespace=\$1
}
kubeconfig ()
{
    TMP_KUBECONFIG=\$(mktemp);
    kubectl get secret admin-kubeconfig -o go-template='{{ index .data "kubeconfig" }}' | base64 -d > \$TMP_KUBECONFIG;
    export KUBECONFIG=\$TMP_KUBECONFIG;
    cn kube-system
}
alias k=kubectl
source <(k completion bash )
source <(k completion bash | sed s/kubectl/k/g)
export KUBECONFIG=~/.kube/kind-config-prow-build-cluster
EOF

# The container runtime allows us to change the content but not to change the inode
# which is what sed -i does, so write to a tempfile and write the tempfiles content back
temp_hosts="$(mktemp)"
sed 's/localhost/localhost dex.oauth/' /etc/hosts > $temp_hosts
cat $temp_hosts >/etc/hosts

# Create kind cluster
TEST_NAME="Create kind cluster"
echodate "Creating the kind cluster"
kind create cluster --name ${SEED_NAME}
cp ~/.kube/kind-config-${SEED_NAME} ~/.kube/config
export KUBECONFIG="$(kind get kubeconfig-path --name=${SEED_NAME})"

function cleanup {
  testRC=$?
  echodate "Starting cleanup"
  set +e

  # Delete all clusters
  kubectl delete cluster --all

  # Upload the JUNIT files
  mv /reports/* ${ARTIFACTS}/
  echodate "Finished cleanup"

}
trap cleanup EXIT

echodate "Setting up iptables rules for to make nodeports available"
iptables -t nat -A PREROUTING -i eth0 -p tcp -m multiport --dports=30000:33000 -j DNAT --to-destination 172.17.0.2
# Docker sets up a MASQUERADE rule for postrouting, so nothing to do for us
echodate "Successfully set up iptables rules for nodeports"

## TODO: Is this needed?
TEST_NAME="Get Vault token"
echodate "Getting secrets from Vault"
export VAULT_ADDR=https://vault.loodse.com/
export VAULT_TOKEN=$(vault write \
  --format=json auth/approle/login \
  role_id=$VAULT_ROLE_ID secret_id=$VAULT_SECRET_ID \
  | jq .auth.client_token -r)

TEST_NAME="Deploying dex"
echodate "Deploying dex"
mkdir -p ${SCRIPT_PATH}/helm
cp -r ${KUBERMATIC_PATH}/config/oauth ${DEX_PATH}
rm ${DEX_PATH}/templates/ingress.yaml
cp ${SCRIPT_PATH}/testdata/oauth_values.yaml ${DEX_PATH}/values.yaml
cp ${SCRIPT_PATH}/testdata/oauth_configmap.yaml ${DEX_PATH}/templates/configmap.yaml
rm ${KUBERMATIC_HELM_PATH}/templates/ingress.yaml

TEST_NAME="Deploying kubermatic CRDs"
retry 5 kubectl apply -f config/kubermatic/crd

helm install --wait --timeout 180 \
  --set-string=dex.ingress.host=http://dex.oauth:5556 \
  --values ${DEX_PATH}/values.yaml \
  --namespace ${DEX_NAMESPACE} \
  --name kubermatic-oauth-e2e ${DEX_PATH}


# Build kubermatic binaries and push the image
echodate "Building containers with tag $KUBERMATIC_VERSION"
echodate "Building binaries"
TEST_NAME="Build Kubermatic binaries"
time retry 1 make -C api build

(
  echodate "Building docker image"
  TEST_NAME="Build Kubermatic Docker image"
  cd api
  IMAGE_NAME="quay.io/kubermatic/api:$KUBERMATIC_VERSION"
  time retry 5 docker build -t "$IMAGE_NAME" .
  time retry 5 kind load docker-image "$IMAGE_NAME" --name ${SEED_NAME}
)
(
  echodate "Building addons image"
  TEST_NAME="Build addons Docker image"
  cd addons
  IMAGE_NAME="quay.io/kubermatic/addons:$KUBERMATIC_VERSION"
  time retry 5 docker build -t "${IMAGE_NAME}" .
  time retry 5 kind load docker-image "$IMAGE_NAME" --name ${SEED_NAME}
)
(
  echodate "Building openshift addons image"
  TEST_NAME="Build openshift Docker image"
  cd openshift_addons
  IMAGE_NAME="quay.io/kubermatic/openshift-addons:$KUBERMATIC_VERSION"
  time retry 5 docker build -t "${IMAGE_NAME}" .
  time retry 5 kind load docker-image "$IMAGE_NAME" --name ${SEED_NAME}
)
(
  echodate "Building dnatcontroller image"
  TEST_NAME="Build dnatcontroller Docker image"
  cd api/cmd/kubeletdnat-controller
  make build
  IMAGE_NAME="quay.io/kubermatic/kubeletdnat-controller:$KUBERMATIC_VERSION"
  time retry 5 docker build -t "${IMAGE_NAME}" .
  time retry 5 kind load docker-image "$IMAGE_NAME" --name ${SEED_NAME}
)
(
  echodate "Building user-ssh-keys-agent image"
  TEST_NAME="Build user-ssh-keys-agent Docker image"
  cd api/cmd/user-ssh-keys-agent
  make build
  IMAGE_NAME=quay.io/kubermatic/user-ssh-keys-agent:$KUBERMATIC_VERSION
  time retry 5 docker build -t "${IMAGE_NAME}" .
  time retry 5 kind load docker-image "$IMAGE_NAME" --name ${SEED_NAME}
)

echodate "Successfully built and loaded all images"

INITIAL_MANIFESTS="$(mktemp)"
cat <<EOF >$INITIAL_MANIFESTS
apiVersion: v1
kind: Namespace
metadata:
  name: kubermatic
spec: {}
status: {}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prow-tiller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: tiller
    namespace: kube-system
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: prow-kubermatic
subjects:
- kind: ServiceAccount
  name: kubermatic
  namespace: kubermatic
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
---
EOF
TEST_NAME="Create Kubermatic bindings"
retry 5 kubectl apply -f $INITIAL_MANIFESTS

echodate "Deploying tiller"
TEST_NAME="Deploy Tiller"
helm init --wait --service-account=tiller

# Defaults to a hardcoded version so we do not test by default if the latest dashboard version
# got successfully built.
LATEST_DASHBOARD="${LATEST_DASHBOARD:-43037e8f118f0e310cfcae713bc2b3bd1a2c8496}"

# --force is needed in case the first attempt at installing didn't succeed
# see https://github.com/helm/helm/pull/3597
retry 3 helm upgrade --install --force --wait --timeout 300 \
  --set=kubermatic.isMaster=true \
  --set-string=kubermatic.controller.addons.kubernetes.image.tag="$KUBERMATIC_VERSION" \
  --set-string=kubermatic.controller.image.tag="$KUBERMATIC_VERSION" \
  --set-string=kubermatic.controller.addons.openshift.image.tag="$KUBERMATIC_VERSION" \
  --set-string=kubermatic.api.image.tag="$KUBERMATIC_VERSION" \
  --set-string=kubermatic.masterController.image.tag="$KUBERMATIC_VERSION" \
  --set-string=kubermatic.ui.image.tag=${LATEST_DASHBOARD} \
  --set=kubermatic.ingressClass=non-existent \
  --set=kubermatic.checks.crd.disable=true \
  --set=kubermatic.datacenters='' \
  --set=kubermatic.dynamicDatacenters=true \
  ${OPENSHIFT_HELM_ARGS:-} \
  --values ${VALUES_FILE} \
  kubermatic \
  ./config/kubermatic/
echodate "Finished installing Kubermatic"

echodate "Installing seeds"
SEED_MANIFEST="$(mktemp)"
cat <<EOF >$SEED_MANIFEST
kind: Secret
apiVersion: v1
metadata:
  name: ${SEED_NAME}-kubeconfig
  namespace: kubermatic
data:
  kubeconfig: "$(cat $KUBECONFIG|base64 -w0)"
---
kind: Seed
apiVersion: kubermatic.k8s.io/v1
metadata:
  name: ${SEED_NAME}
  namespace: kubermatic
  labels:
    worker-name: "$BUILD_ID"
spec:
  country: Germany
  location: Hamburg
  kubeconfig:
    name: ${SEED_NAME}-kubeconfig
    namespace: kubermatic
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
EOF
TEST_NAME="Deploy Seed Manifest"
retry 7 kubectl apply -f $SEED_MANIFEST
echodate "Finished installing seed"
