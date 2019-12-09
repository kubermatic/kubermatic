#############################################################
## CI Setup Kubermatic in kind                              #
## A simple script to get a Kubermatic setup using kind     #
#############################################################
#
# This script should be sourced, not called, so callers get the variables it sets

# The kubemaric version to build
export KUBERMATIC_VERSION="${KUBERMATIC_VERSION:-$(git rev-parse HEAD)}"
export SEED_NAME=prow-build-cluster
export KUBERMATIC_APISERVER_ADDRESS="localhost:8080"
export KUBERMATIC_NO_WORKER_NAME=true
export KUBERMATIC_SKIP_BUILDING="${KUBERMATIC_SKIP_BUILDING:-false}"
# Number of UI replicas, zero by default as we do not test the UI
export KUBERMATIC_UI_REPLICAS="${KUBERMATIC_UI_REPLICAS:-0}"
# Defaults to a hardcoded version so we do not test by default if the latest dashboard version
# got successfully built.
export KUBERMATIC_DASHBOARD_VERSION="${KUBERMATIC_DASHBOARD_VERSION:-latest}"

# Consider self-installed go installations
export PATH=$PATH:/usr/local/go/bin

if [[ -z ${JOB_NAME} ]]; then
	echo "This script should only be running in a CI environment."
	exit 1
fi

if [[ -z ${PROW_JOB_ID} ]]; then
	echo "Build id env variable has to be set."
	exit 1
fi

cd "${GOPATH}/src/github.com/kubermatic/kubermatic"
source ./api/hack/lib.sh

TEST_NAME="Get Vault token"
echodate "Getting secrets from Vault"
export VAULT_ADDR=https://vault.loodse.com/
export VAULT_TOKEN=$(vault write \
  --format=json auth/approle/login \
  role_id=$VAULT_ROLE_ID secret_id=$VAULT_SECRET_ID \
  | jq .auth.client_token -r)
export VALUES_FILE=/tmp/values.yaml
TEST_NAME="Get Values file from Vault"
retry 5 vault kv get -field=values.yaml \
  dev/seed-clusters/ci.kubermatic.io > $VALUES_FILE

# Set docker config
echo $IMAGE_PULL_SECRET_DATA | base64 -d > /config.json

# Start docker daemon
echodate "Starting docker"
dockerd > /tmp/docker.log 2>&1 &
echodate "Started docker"

function docker_logs {
  originalRC=$?
  if [[ $originalRC -ne 0 ]]; then
    echodate "Printing docker logs"
    cat /tmp/docker.log
    echodate "Done printing docker logs"
  fi
  return $originalRC
}
trap docker_logs EXIT

# Wait for it to start
echodate "Waiting for docker"
retry 5 docker stats --no-stream
echodate "Docker became ready"

# Load kind image
echodate "Loading kindest image"
docker load --input /kindest.tar
echodate "Loaded kindest image"

# Prevent mtu-related timeouts
echodate "Setting iptables rule to clamp mss to path mtu"
iptables -t mangle -A POSTROUTING -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu
echodate "Set iptables rule to clamp mss to path mtu"


# Make debugging a bit better
echodate "Setting aliases in .bashrc"
cat <<EOF >>~/.bashrc
# Gets set to the CI clusters kubeconfig from a preset
unset KUBECONFIG
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
EOF
echodate "Set aliases in .bashrc"

# The container runtime allows us to change the content but not to change the inode
# which is what sed -i does, so write to a tempfile and write the tempfiles content back
echodate "Setting dex.oauth alias in /etc/hosts"
temp_hosts="$(mktemp)"
sed 's/localhost/localhost dex.oauth/' /etc/hosts > $temp_hosts
# I will regret this...
echo '10.98.184.166 minio.gocache.svc.cluster.local.' >> $temp_hosts
cat $temp_hosts >/etc/hosts
echodate "Set dex.oauth alias in /etc/hosts"

# Create kind cluster
TEST_NAME="Create kind cluster"
echodate "Creating the kind cluster"
export KUBECONFIG=~/.kube/config
kind create cluster --name ${SEED_NAME} --image=kindest/node:v1.15.6

echodate "Starting clusterexposer"
make -C api download-gocache
CGO_ENABLED=0 go run ./api/pkg/test/clusterexposer/cmd \
  --kubeconfig-inner "$KUBECONFIG" \
  --kubeconfig-outer "/etc/kubeconfig/kubeconfig" \
  --build-id "$PROW_JOB_ID" &> /var/log/clusterexposer.log &

function print_cluster_exposer_logs {
  originalRC=$?

  # Tolerate errors and just continue
  set +e
  echodate "Printing clusterexposer logs"
  cat /var/log/clusterexposer.log
  echodate "Done printing clusterexposer logs"
  set -e

  return $originalRC
}
trap print_cluster_exposer_logs EXIT

TEST_NAME="Wait for cluster exposer"
echodate "Waiting for cluster exposer to be running"

retry 5 curl --fail http://127.0.0.1:2047/metrics
echodate "Cluster exposer is running"

echodate "Setting up iptables rules for to make nodeports available"
iptables -t nat -A PREROUTING -i eth0 -p tcp -m multiport --dports=30000:33000 -j DNAT --to-destination 172.17.0.2
# Docker sets up a MASQUERADE rule for postrouting, so nothing to do for us
echodate "Successfully set up iptables rules for nodeports"

INITIAL_MANIFESTS="$(mktemp)"
cat <<EOF >$INITIAL_MANIFESTS
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: tiller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: tiller
    namespace: kube-system
---
EOF
TEST_NAME="Create Helm bindings"
retry 5 kubectl apply -f $INITIAL_MANIFESTS

echodate "Deploying tiller"
TEST_NAME="Deploy Tiller"
helm init --wait --service-account=tiller

TEST_NAME="Deploying dex"
echodate "Deploying dex"
rm config/oauth/templates/ingress.yaml
cp $(dirname $0)/testdata/oauth_configmap.yaml config/oauth/templates/configmap.yaml

echodate "Creating kubermatic-fast storageclass"
TEST_NAME="Create kubermatic-fast storageclass"
retry 5 kubectl get storageclasses.storage.k8s.io standard -o json \
  |jq 'del(.metadata)|.metadata.name = "kubermatic-fast"'\
  |kubectl apply -f -
echodate "Successfully created kubermatic-fast storageclass"

helm install --wait --timeout 180 \
  --set-string=dex.ingress.host=http://dex.oauth:5556 \
  --values ./api/hack/ci/testdata/oauth_values.yaml \
  --namespace oauth \
  --name kubermatic-oauth-e2e ./config/oauth

TEST_NAME="Deploying kubermatic CRDs"
retry 5 kubectl apply -f config/kubermatic/crd

if [[ "${KUBERMATIC_SKIP_BUILDING}" = "false" ]]; then
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
    retry 5 docker login -u "$QUAY_IO_USERNAME" -p "$QUAY_IO_PASSWORD" quay.io
    IMAGE_NAME=quay.io/kubermatic/user-ssh-keys-agent:$KUBERMATIC_VERSION
    time retry 5 docker build -t "${IMAGE_NAME}" .
    time retry 5 docker push "quay.io/kubermatic/user-ssh-keys-agent:$KUBERMATIC_VERSION"
  )
  echodate "Successfully built and loaded all images"
fi


# --force is needed in case the first attempt at installing didn't succeed
# see https://github.com/helm/helm/pull/3597
retry 3 helm upgrade --install --force --wait --timeout 300 \
  --set=kubermatic.isMaster=true \
  --set=kubermatic.imagePullSecretData=$IMAGE_PULL_SECRET_DATA \
  --set-string=kubermatic.controller.addons.kubernetes.image.tag="$KUBERMATIC_VERSION" \
  --set-string=kubermatic.controller.image.tag="$KUBERMATIC_VERSION" \
  --set-string=kubermatic.controller.addons.openshift.image.tag="$KUBERMATIC_VERSION" \
  --set-string=kubermatic.api.image.tag="$KUBERMATIC_VERSION" \
  --set=kubermatic.controller.datacenterName=${SEED_NAME} \
  --set=kubermatic.api.replicas=1 \
  --set-string=kubermatic.masterController.image.tag="$KUBERMATIC_VERSION" \
  --set-string=kubermatic.ui.image.tag=${KUBERMATIC_DASHBOARD_VERSION} \
  --set=kubermatic.ui.replicas="${KUBERMATIC_UI_REPLICAS}" \
  --set=kubermatic.ingressClass=non-existent \
  --set=kubermatic.checks.crd.disable=true \
  --set=kubermatic.datacenters='' \
  --set=kubermatic.dynamicDatacenters=true \
  --set=kubermatic.kubeconfig="$(cat $KUBECONFIG|sed 's/127.0.0.1.*/kubernetes.default.svc.cluster.local./'|base64 -w0)" \
  --set=kubermatic.auth.tokenIssuer=http://dex.oauth:5556 \
  --set=kubermatic.auth.clientID=kubermatic \
  --set=kubermatic.auth.serviceAccountKey=$SERVICE_ACCOUNT_KEY \
  --set=kubermatic.apiserverDefaultReplicas=1 \
  --set=kubermatic.deployVPA=false \
  --namespace=kubermatic \
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
  kubeconfig: "$(cat $KUBECONFIG|sed 's/127.0.0.1.*/kubernetes.default.svc.cluster.local./'|base64 -w0)"
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

# Expose dex to localhost
TEST_NAME="Expose dex and kubermatic API to localhost"
kubectl port-forward --address 0.0.0.0 -n oauth svc/dex 5556  &

# Expose kubermatic API to localhost
kubectl port-forward --address 0.0.0.0 -n kubermatic svc/kubermatic-api 8080:80 &
echodate "Finished exposing components"

echodate "Waiting for dex to be ready"
retry 5 curl --fail http://127.0.0.1:5556/healthz
echodate "Dex got ready"

echodate "Waiting for api to be ready"
retry 5 curl --fail http://127.0.0.1:8080/api/v1/healthz
echodate "API got ready"

function cleanup_kubermatic_clusters_in_kind {
  originalRC=$?

  # Tolerate errors and just continue
  set +e
  # Clean up clusters
  kubectl delete cluster --all --ignore-not-found=true

  # Kill all descendant processes
  pkill -P $$
  set -e

  return $originalRC
}
trap cleanup_kubermatic_clusters_in_kind EXIT

cd -
