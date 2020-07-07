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

#############################################################
## CI Setup Kubermatic in kind                              #
## A simple script to get a Kubermatic setup using kind     #
#############################################################
#
# This script should be sourced, not called, so callers get the variables it sets
# Note: This script is used in upgrade test, hence it must be idempotent

# The kubemaric version to build. The script will checkout the configured
# revision before building the binaries and go back to the initial HEAD after
# building finished. Defaults to current HEAD.
export KUBERMATIC_VERSION="${KUBERMATIC_VERSION:-$(git rev-parse HEAD)}"
export KUBERMATIC_SCHEME="http"
export KUBERMATIC_HOST="localhost:8080"
# If set to `true`, the script will just use `latest`. Used e.G. in the UI tests.
export KUBERMATIC_SKIP_BUILDING="${KUBERMATIC_SKIP_BUILDING:-false}"
# If set to `true`, the script will use the Kubermatic Operator instead of the
# Helm chart for setting up Kubermatic.
export KUBERMATIC_USE_OPERATOR="${KUBERMATIC_USE_OPERATOR:-false}"
# Number of UI replicas, zero by default as we do not test the UI
export KUBERMATIC_UI_REPLICAS="${KUBERMATIC_UI_REPLICAS:-0}"
# Defaults to `latest` so we do not test by default if the latest dashboard version
# got successfully built and published, as that may race with the dashboard postsubmit.
export UIDOCKERTAG="${UIDOCKERTAG:-latest}"
# ADDITIONAL_HELM_ARGS allows to configure extra args for helm. Used e.G. in UI and API tests.
export ADDITIONAL_HELM_ARGS="${ADDITIONAL_HELM_ARGS:-}"

# Consider self-installed go installations
export PATH=$PATH:/usr/local/go/bin

# This is just used as a const
export SEED_NAME=prow-build-cluster

if [[ -z ${JOB_NAME} ]]; then
	echo "This script should only be running in a CI environment."
	exit 1
fi

if [[ -z ${PROW_JOB_ID} ]]; then
	echo "Build id env variable has to be set."
	exit 1
fi

cd "${GOPATH}/src/github.com/kubermatic/kubermatic"
source hack/lib.sh

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
if ps xf| grep -v grep | grep -q dockerd; then
  echodate "Docker already started"
else
  echodate "Starting docker"
  dockerd > /tmp/docker.log 2>&1 &
  echodate "Started docker"
fi

function docker_logs {
  originalRC=$?
  if [[ $originalRC -ne 0 ]]; then
    echodate "Printing docker logs"
    cat /tmp/docker.log
    echodate "Done printing docker logs"
  fi
  return $originalRC
}
appendTrap docker_logs EXIT

# Wait for it to start
echodate "Waiting for docker"
retry 5 docker stats --no-stream
echodate "Docker became ready"

# Load kind image
if docker images | grep -q kindest; then
  echodate "Kindest image already loaded"
else
  echodate "Loading kindest image"
  docker load --input /kindest.tar
  echodate "Loaded kindest image"
fi

# Prevent mtu-related timeouts
echodate "Setting iptables rule to clamp mss to path mtu"
iptables -t mangle -A POSTROUTING -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu

# Make debugging a bit better
echodate "Confuguring bash"
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
# this alias makes it so that watch can be used with other aliases, like "watch k get pods"
alias watch='watch '
alias k=kubectl
alias ll='ls -lh --file-type --group-directories-first'
alias lll='ls -lahF --group-directories-first'
source <(k completion bash )
source <(k completion bash | sed s/kubectl/k/g)
EOF

# The container runtime allows us to change the content but not to change the inode
# which is what sed -i does, so write to a tempfile and write the tempfiles content back.
# The alias makes it easier to access the port-forwarded Dex inside the Kind cluster;
# the token issuer cannot be localhost:5556, because pods inside the cluster would not
# find Dex anymore.
echodate "Setting dex.oauth alias in /etc/hosts"
temp_hosts="$(mktemp)"
sed 's/localhost/localhost dex.oauth/' /etc/hosts > $temp_hosts
cat $temp_hosts >/etc/hosts
echodate "Set dex.oauth alias in /etc/hosts"

if kind get clusters | grep -q ${SEED_NAME}; then
  echodate "Kind cluster already exists"
else
  # Create kind cluster
  TEST_NAME="Create kind cluster"
  echodate "Creating the kind cluster"
  export KUBECONFIG=~/.kube/config

  beforeKindCreate=$(nowms)
  nodeVersion=v1.15.6
  kind create cluster --name ${SEED_NAME} --image=kindest/node:$nodeVersion
  pushElapsed kind_cluster_create_duration_milliseconds $beforeKindCreate "node_version=\"$nodeVersion\""
fi

if ls /var/log/clusterexposer.log &>/dev/null; then
  echodate "Cluster-Exposer already running"
else
  echodate "Starting clusterexposer"

  beforeGocache=$(nowms)
  make download-gocache
  pushElapsed gocache_download_duration_milliseconds $beforeGocache

  CGO_ENABLED=0 go build --tags "$KUBERMATIC_EDITION" -v -o /tmp/clusterexposer ./pkg/test/clusterexposer/cmd
  CGO_ENABLED=0 /tmp/clusterexposer \
    --kubeconfig-inner "$KUBECONFIG" \
    --kubeconfig-outer "/etc/kubeconfig/kubeconfig" \
    --build-id "$PROW_JOB_ID" &> /var/log/clusterexposer.log &
fi

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
appendTrap print_cluster_exposer_logs EXIT

TEST_NAME="Wait for cluster exposer"
echodate "Waiting for cluster exposer to be running"

retry 5 curl --fail http://127.0.0.1:2047/metrics -o /dev/null
echodate "Cluster exposer is running"

echodate "Setting up iptables rules for to make nodeports available"
iptables -t nat -A PREROUTING -i eth0 -p tcp -m multiport --dports=30000:33000 -j DNAT --to-destination 172.17.0.2
# By default all traffic gets dropped unless specified (tested with docker
# server 18.09.1)
iptables -t filter -I DOCKER-USER -d 172.17.0.2/32 ! -i docker0 -o docker0 -p tcp -m multiport --dports=30000:33000 -j ACCEPT

# Docker sets up a MASQUERADE rule for postrouting, so nothing to do for us
echodate "Successfully set up iptables rules for nodeports"

echodate "Creating kubermatic-fast storageclass"
TEST_NAME="Create kubermatic-fast storageclass"
retry 5 kubectl get storageclasses.storage.k8s.io standard -o json \
  |jq 'del(.metadata)|.metadata.name = "kubermatic-fast"'\
  |kubectl apply -f -
echodate "Successfully created kubermatic-fast storageclass"

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
echodate "Creating Helm bindings"
retry 5 kubectl apply -f $INITIAL_MANIFESTS

TEST_NAME="Deploy Tiller"
if kubectl get deployment -n kube-system tiller-deploy &>/dev/null; then
  echodate "Tiller is already deployed."
else
  echodate "Deploying Tiller"
  helm init --wait --service-account=tiller
fi

TEST_NAME="Deploy Dex"
echodate "Deploying Dex"

export KUBERMATIC_DEX_VALUES_FILE=$(realpath hack/ci/testdata/oauth_values.yaml)

if kubectl get namespace oauth; then
  echodate "Dex already deployed"
else
  retry 5 helm install --wait --timeout 180 \
    --values $KUBERMATIC_DEX_VALUES_FILE \
    --namespace oauth \
    --name oauth charts/oauth/
fi

export KUBERMATIC_OIDC_LOGIN="roxy@loodse.com"
export KUBERMATIC_OIDC_PASSWORD="password"

TEST_NAME="Deploy Kubermatic CRDs"
echodate "Deploying Kubermatic CRDs"

retry 5 kubectl apply -f charts/kubermatic/crd/

REPOSUFFIX=""
if [ "$KUBERMATIC_EDITION" != "ce" ]; then
  REPOSUFFIX="-$KUBERMATIC_EDITION"
fi

if [[ "${KUBERMATIC_SKIP_BUILDING}" = "false" ]]; then
  # Build kubermatic binaries and push the image
  OLD_HEAD="$(git rev-parse HEAD)"
  git checkout ${KUBERMATIC_VERSION}
  echodate "Building containers with tag $KUBERMATIC_VERSION"
  echodate "Building binaries"
  TEST_NAME="Build Kubermatic binaries"

  beforeGoBuild=$(nowms)
  time retry 1 make build
  pushElapsed kubermatic_go_build_duration_milliseconds $beforeGoBuild

  beforeDockerBuild=$(nowms)

  (
    echodate "Building docker image"
    TEST_NAME="Build Kubermatic Docker image"
    IMAGE_NAME="quay.io/kubermatic/kubermatic$REPOSUFFIX:$KUBERMATIC_VERSION"
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
    # The operator automatically creates a nodeport-proxy, so it needs the
    # Docker image to exist.
    if [[ "${KUBERMATIC_USE_OPERATOR}" = "true" ]]; then
      echodate "Building nodeport-proxy image"
      TEST_NAME="Build nodeport-proxy Docker image"
      cd cmd/nodeport-proxy
      make build
      IMAGE_NAME="quay.io/kubermatic/nodeport-proxy:$KUBERMATIC_VERSION"
      time retry 5 docker build -t "${IMAGE_NAME}" .
      time retry 5 kind load docker-image "$IMAGE_NAME" --name ${SEED_NAME}
    fi
  )
  (
    echodate "Building dnatcontroller image"
    TEST_NAME="Build dnatcontroller Docker image"
    cd cmd/kubeletdnat-controller
    make build
    IMAGE_NAME="quay.io/kubermatic/kubeletdnat-controller:$KUBERMATIC_VERSION"
    time retry 5 docker build -t "${IMAGE_NAME}" .
    time retry 5 kind load docker-image "$IMAGE_NAME" --name ${SEED_NAME}
  )
  (
    echodate "Building user-ssh-keys-agent image"
    TEST_NAME="Build user-ssh-keys-agent Docker image"
    cd cmd/user-ssh-keys-agent
    make build
    retry 5 docker login -u "$QUAY_IO_USERNAME" -p "$QUAY_IO_PASSWORD" quay.io
    IMAGE_NAME=quay.io/kubermatic/user-ssh-keys-agent:$KUBERMATIC_VERSION
    time retry 5 docker build -t "${IMAGE_NAME}" .
    time retry 5 docker push "${IMAGE_NAME}"
  )
  (
    echodate "Building etcd-launcher images"
    TEST_NAME="Build etcd-launcher Docker images"

    for ETCD_TAG in $(getEtcdTags); do
      BASE_TAG=$(echo ${ETCD_TAG} | cut -d\. -f 1,2| tr -d .)
      IMAGE_NAME="quay.io/kubermatic/etcd-launcher-${BASE_TAG}:${KUBERMATIC_VERSION}"
      time retry 5 docker build --build-arg ETCD_VERSION=${ETCD_TAG} -t "${IMAGE_NAME}" -f cmd/etcd-launcher/Dockerfile .
      time retry 5 kind load docker-image "$IMAGE_NAME" --name ${SEED_NAME}
    done
  )

  pushElapsed kubermatic_docker_build_duration_milliseconds $beforeDockerBuild

  git checkout ${OLD_HEAD}
  echodate "Successfully built and loaded all images"
fi

function check_all_deployments_ready() {
  local namespace="$1"

  # check that Deployments have been created
  local deployments
  deployments=$(kubectl -n $namespace get deployments -o json)

  if [ $(jq '.items | length' <<< $deployments) -eq 0 ]; then
    echodate "No Deployments created yet."
    return 1
  fi

  # check that all Deployments are ready
  local unready
  unready=$(jq -r '[.items[] | select(.spec.replicas > 0) | select (.status.availableReplicas < .spec.replicas) | .metadata.name] | @tsv' <<< $deployments)
  if [ -n "$unready" ]; then
    echodate "Not all Deployments have finished rolling out, namely: $unready"
    return 1
  fi

  return 0
}

# We don't need a valid certificate (and can't even get one), but still need
# to have the CRDs installed so we can at least create a Certificate resource.
TEST_NAME="Deploy cert-manager CRDs"
echodate "Deploying cert-manager CRDs"
retry 5 kubectl apply -f charts/cert-manager/crd/

if [[ "${KUBERMATIC_USE_OPERATOR}" = "false" ]]; then
  TEST_NAME="Deploy Kubermatic"
  echodate "Deploying Kubermatic using Helm..."

  OLD_HEAD="$(git rev-parse HEAD)"
  if [[ -n ${CHARTS_VERSION:-} ]]; then
    git checkout "$CHARTS_VERSION"
  fi

  beforeDeployment=$(nowms)

  # --force is needed in case the first attempt at installing didn't succeed
  # see https://github.com/helm/helm/pull/3597;
  # we always override the quay repositories so we don't have to care if the
  # Helm chart is made for CE or EE
  retry 3 helm upgrade --install --force --wait --timeout 300 \
    --set=kubermatic.isMaster=true \
    --set=kubermatic.imagePullSecretData=$IMAGE_PULL_SECRET_DATA \
    --set-string=kubermatic.controller.image.repository="quay.io/kubermatic/kubermatic$REPOSUFFIX" \
    --set-string=kubermatic.masterController.image.repository="quay.io/kubermatic/kubermatic$REPOSUFFIX" \
    --set-string=kubermatic.api.image.repository="quay.io/kubermatic/kubermatic$REPOSUFFIX" \
    --set-string=kubermatic.ui.image.repository="quay.io/kubermatic/dashboard$REPOSUFFIX" \
    --set-string=kubermatic.controller.addons.kubernetes.image.tag="$KUBERMATIC_VERSION" \
    --set-string=kubermatic.controller.image.tag="$KUBERMATIC_VERSION" \
    --set-string=kubermatic.controller.addons.openshift.image.tag="$KUBERMATIC_VERSION" \
    --set-string=kubermatic.api.image.tag="$KUBERMATIC_VERSION" \
    --set=kubermatic.controller.datacenterName=${SEED_NAME} \
    --set=kubermatic.api.replicas=1 \
    --set-string=kubermatic.masterController.image.tag="$KUBERMATIC_VERSION" \
    --set-string=kubermatic.ui.image.tag=${UIDOCKERTAG} \
    --set=kubermatic.ui.replicas="${KUBERMATIC_UI_REPLICAS}" \
    --set=kubermatic.ingressClass=non-existent \
    --set=kubermatic.checks.crd.disable=true \
    --set=kubermatic.datacenters='' \
    --set=kubermatic.dynamicDatacenters=true \
    --set=kubermatic.dynamicPresets=true \
    --set=kubermatic.kubeconfig="$(cat $KUBECONFIG|sed 's/127.0.0.1.*/kubernetes.default.svc.cluster.local./'|base64 -w0)" \
    --set=kubermatic.auth.tokenIssuer=http://dex.oauth:5556/dex \
    --set=kubermatic.auth.clientID=kubermatic \
    --set=kubermatic.auth.serviceAccountKey=$SERVICE_ACCOUNT_KEY \
    --set=kubermatic.apiserverDefaultReplicas=1 \
    --set=kubermatic.deployVPA=false \
    --namespace=kubermatic \
    ${ADDITIONAL_HELM_ARGS} \
    ${OPENSHIFT_HELM_ARGS:-} \
    --values ${VALUES_FILE} \
    kubermatic \
    charts/kubermatic/

  pushElapsed kubermatic_deployment_duration_milliseconds $beforeDeployment 'method="helm"'

  # Return repo to previous state if we checked out older charts before.
  if [[ "${KUBERMATIC_SKIP_BUILDING}" = "false" ]]; then
    git checkout ${KUBERMATIC_VERSION}
  fi
else
  TEST_NAME="Deploy Kubermatic"
  echodate "Installing Kubermatic using operator..."

  beforeDeployment=$(nowms)

  # --force is needed in case the first attempt at installing didn't succeed
  # see https://github.com/helm/helm/pull/3597
  retry 3 helm upgrade --install --force --wait --timeout 300 \
    --set-file "kubermaticOperator.imagePullSecret=/config.json" \
    --set-string "kubermaticOperator.image.repository=quay.io/kubermatic/kubermatic$REPOSUFFIX" \
    --set-string "kubermaticOperator.image.tag=$KUBERMATIC_VERSION" \
    --namespace kubermatic \
    --values ${VALUES_FILE} \
    kubermatic-operator \
    charts/kubermatic-operator/

  echodate "Kubermatic Operator is ready."

  # No need to override any Docker repositories here, as the operator
  # is already pinned to CE or EE and has the proper default values on board.

  KUBERMATIC_CONFIG="$(mktemp)"
  cat <<EOF >$KUBERMATIC_CONFIG
apiVersion: operator.kubermatic.io/v1alpha1
kind: KubermaticConfiguration
metadata:
  name: e2e
  namespace: kubermatic
spec:
  ingress:
    domain: ci.kubermatic.io
    disable: true
  imagePullSecret: |
$(echo "$IMAGE_PULL_SECRET_DATA" | base64 -d | sed 's/^/    /')
  userCluster:
    apiserverReplicas: 1
  api:
    debugLog: true
  featureGates:
    # VPA won't do anything useful due to missing Prometheus, but we can
    # at least ensure we deploy a working set of Deployments.
    VerticalPodAutoscaler: {}
  ui:
    replicas: $KUBERMATIC_UI_REPLICAS
  # Dex integration
  auth:
    tokenIssuer: "http://dex.oauth:5556/dex"
    clientID: "kubermatic"
    issuerRedirectURL: "http://localhost:8000"
    serviceAccountKey: "$SERVICE_ACCOUNT_KEY"
EOF
  echodate "Creating Kubermatic Master..."
  retry 3 kubectl apply -f $KUBERMATIC_CONFIG

  echodate "Waiting for Kubermatic Operator to deploy master components..."
  # sleep a bit to prevent us from checking the Deployments too early, before
  # the operator had time to reconcile
  sleep 5
  retry 10 check_all_deployments_ready kubermatic

  echodate "Kubermatic Master is ready."

  pushElapsed kubermatic_deployment_duration_milliseconds $beforeDeployment 'method="operator"'
fi

echodate "Finished installing Kubermatic"

echodate "Installing Seed..."
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
    byo-kubernetes:
      location: Frankfurt
      country: DE
      spec:
         bringyourown: {}
    alibaba-eu-central-1a:
      location: Frankfurt
      country: DE
      spec:
        alibaba:
          region: eu-central-1
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
    do-fra1:
      location: Frankfurt
      country: DE
      spec:
        digitalocean:
          region: fra1
    kubevirt-europe-west3-c:
      location: Frankfurt
      country: DE
      spec:
        kubevirt: {}
    syseleven-dbl1:
      country: DE
      location: Syseleven - dbl1
      spec:
        openstack:
          auth_url: https://api.cbk.cloud.syseleven.net:5000/v3
          availability_zone: dbl1
          dns_servers:
          - 37.123.105.116
          - 37.123.105.117
          enforce_floating_ip: true
          ignore_volume_az: false
          images:
            centos: kubermatic-e2e-centos
            coreos: kubermatic-e2e-coreos
            ubuntu: kubermatic-e2e-ubuntu
          node_size_requirements:
            minimum_memory: 0
            minimum_vcpus: 0
          region: dbl
EOF
retry 8 kubectl apply -f $SEED_MANIFEST
echodate "Finished installing Seed"

# wait until the operator has reconciled
if [[ "${KUBERMATIC_USE_OPERATOR}" = "true" ]]; then
  sleep 5
  echodate "Waiting for Kubermatic Operator to deploy seed components..."
  retry 8 check_all_deployments_ready kubermatic
  echodate "Kubermatic Seed is ready."

  echodate "Waiting for VPA to be ready..."
  retry 5 check_all_deployments_ready kube-system
  echodate "VPA is ready."
fi

function kill_port_forwardings() {
  echodate "Stopping any previous port-forwardings to port $1..."
  ss -tlpn "sport = :$1" | (grep -oP "(?<=pid=)[0-9]+" || true) | uniq | tee | xargs -r kill
}

function cleanup_kubermatic_clusters_in_kind {
  originalRC=$?

  # Tolerate errors and just continue
  set +e
  # Clean up clusters
  echodate "Cleaning up clusters..."
  kubectl delete cluster --all --ignore-not-found=true
  echodate "Done cleaning up clusters"

  # Kill all descendant processes
  pkill -P $$
  set -e

  return $originalRC
}
appendTrap cleanup_kubermatic_clusters_in_kind EXIT

TEST_NAME="Expose Dex and Kubermatic API"
echodate "Exposing Dex and Kubermatic API to localhost..."
kill_port_forwardings 5556
kill_port_forwardings 8080
kubectl port-forward --address 0.0.0.0 -n oauth svc/dex 5556 &
kubectl port-forward --address 0.0.0.0 -n kubermatic svc/kubermatic-api 8080:80 &
echodate "Finished exposing components"

echodate "Waiting for Dex to be ready"
retry 5 curl -sSf  http://127.0.0.1:5556/dex/healthz
echodate "Dex became ready"

echodate "Waiting for Kubermatic API to be ready"
retry 5 curl -sSf  http://127.0.0.1:8080/api/v1/healthz
echodate "API became ready"

cd -
