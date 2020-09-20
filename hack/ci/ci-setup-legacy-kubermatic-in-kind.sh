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

cd $(dirname $0)/../..
source hack/lib.sh

#############################################################
## CI Setup Kubermatic in kind using legacy Helm chart      #
## A simple script to get a Kubermatic setup using kind     #
## Compared to the ci-setup-kubermatic-in-kind.sh, this     #
## script does not use the installer, but tests the         #
## deprecated `kubermatic` Helm chart.                      #
#############################################################

# This script should be sourced, not called, so callers get the variables it sets

if [ -z "${JOB_NAME:-}" ]; then
  echodate "This script should only be running in a CI environment."
  exit 1
fi

if [ -z "${PROW_JOB_ID:-}" ]; then
  echodate "Build id env variable has to be set."
  exit 1
fi

# The kubemaric version to build.
export KUBERMATIC_VERSION="${KUBERMATIC_VERSION:-$(git rev-parse HEAD)}"

REPOSUFFIX=""
if [ "$KUBERMATIC_EDITION" != "ce" ]; then
  REPOSUFFIX="-$KUBERMATIC_EDITION"
fi

# This is just used as a const
# NB: The CE requires Seeds to be named this way
export SEED_NAME=kubermatic

# This defines the Kubermatic API endpoint the e2e tests will communicate with.
# The api service is kubectl-proxied later on.
export KUBERMATIC_SCHEME="http"
export KUBERMATIC_HOST="localhost:8080"

# Tell the conformance tester what dummy account we configure for the e2e tests.
export KUBERMATIC_DEX_VALUES_FILE=$(realpath hack/ci/testdata/oauth_values.yaml)
export KUBERMATIC_OIDC_LOGIN="roxy@loodse.com"
export KUBERMATIC_OIDC_PASSWORD="password"

# Set docker config
echo "$IMAGE_PULL_SECRET_DATA" | base64 -d > /config.json

# Start Docker daemon
echodate "Starting Docker"
dockerd > /tmp/docker.log 2>&1 &
echodate "Started Docker successfully"

function docker_logs {
  if [[ $? -ne 0 ]]; then
    echodate "Printing Docker logs"
    cat /tmp/docker.log
    echodate "Done printing Docker logs"
  fi
}
appendTrap docker_logs EXIT

# Wait for Docker to start
echodate "Waiting for Docker"
retry 5 docker stats --no-stream
echodate "Docker became ready"

# Load kind image
echodate "Loading kindest image"
docker load --input /kindest.tar
echodate "Loaded kindest image"

# Prevent mtu-related timeouts
echodate "Setting iptables rule to clamp mss to path mtu"
iptables -t mangle -A POSTROUTING -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu

# Make debugging a bit better
echodate "Confuguring bash"
cat <<EOF >>~/.bashrc
# Gets set to the CI clusters kubeconfig from a preset
unset KUBECONFIG

cn() {
  kubectl config set-context \$(kubectl config current-context) --namespace=\$1
}

kubeconfig() {
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

# The alias makes it easier to access the port-forwarded Dex inside the Kind cluster;
# the token issuer cannot be localhost:5556, because pods inside the cluster would not
# find Dex anymore.
echodate "Setting dex.oauth alias in /etc/hosts"
# The container runtime allows us to change the content but not to change the inode
# which is what sed -i does, so write to a tempfile and write the tempfiles content back.
temp_hosts="$(mktemp)"
sed 's/localhost/localhost dex.oauth/' /etc/hosts > $temp_hosts
cat $temp_hosts > /etc/hosts
echodate "Set dex.oauth alias in /etc/hosts"

# Create kind cluster
TEST_NAME="Create kind cluster"
echodate "Creating the kind cluster"
export KUBECONFIG=~/.kube/config

beforeKindCreate=$(nowms)
nodeVersion=v1.18.2
kind create cluster --name ${SEED_NAME} --image=kindest/node:$nodeVersion
pushElapsed kind_cluster_create_duration_milliseconds $beforeKindCreate "node_version=\"$nodeVersion\""

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

# Start cluster exposer, which will expose services from within kind as
# a NodePort service on the host
echodate "Starting cluster exposer"

CGO_ENABLED=0 go build --tags "$KUBERMATIC_EDITION" -v -o /tmp/clusterexposer ./pkg/test/clusterexposer/cmd
CGO_ENABLED=0 /tmp/clusterexposer \
  --kubeconfig-inner "$KUBECONFIG" \
  --kubeconfig-outer "/etc/kubeconfig/kubeconfig" \
  --build-id "$PROW_JOB_ID" &> /var/log/clusterexposer.log &

function print_cluster_exposer_logs {
  if [[ $? -ne 0 ]]; then
    # Tolerate errors and just continue
    set +e
    echodate "Printing cluster exposer logs"
    cat /var/log/clusterexposer.log
    echodate "Done printing cluster exposer logs"
    set -e
  fi
}
appendTrap print_cluster_exposer_logs EXIT

TEST_NAME="Wait for cluster exposer"
echodate "Waiting for cluster exposer to be running"

retry 5 curl -s --fail http://127.0.0.1:2047/metrics -o /dev/null
echodate "Cluster exposer is running"

echodate "Setting up iptables rules for to make nodeports available"
KIND_NETWORK_IF=$(ip -br addr | grep -- 'br-' | cut -d' ' -f1)

iptables -t nat -A PREROUTING -i eth0 -p tcp -m multiport --dports=30000:33000 -j DNAT --to-destination 172.18.0.2
# By default all traffic gets dropped unless specified (tested with docker
# server 18.09.1)
iptables -t filter -I DOCKER-USER -d 172.18.0.2/32 ! -i $KIND_NETWORK_IF -o $KIND_NETWORK_IF -p tcp -m multiport --dports=30000:33000 -j ACCEPT

# Docker sets up a MASQUERADE rule for postrouting, so nothing to do for us
echodate "Successfully set up iptables rules for nodeports"

echodate "Creating StorageClass kubermatic-fast "
TEST_NAME="Create StorageClass kubermatic-fast"
retry 5 kubectl get storageclasses.storage.k8s.io standard -o json \
  | jq 'del(.metadata)|.metadata.name = "kubermatic-fast"'\
  | kubectl apply -f -
echodate "Successfully created StorageClass"

TEST_NAME="Deploy Dex"
echodate "Deploying Dex"

retry 3 kubectl create ns oauth
retry 5 helm3 --namespace oauth install --atomic --timeout 3m --values $KUBERMATIC_DEX_VALUES_FILE oauth charts/oauth/

TEST_NAME="Deploy Kubermatic CRDs"
echodate "Deploying Kubermatic CRDs"

retry 5 kubectl apply -f charts/kubermatic/crd/

# Build binaries and load the Docker images into the kind cluster
echodate "Building binaries for $KUBERMATIC_VERSION"
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
  echodate "Building kubeletdnat-controller image"
  TEST_NAME="Build kubeletdnat-controller Docker image"
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
  echodate "Building etcd-launcher image"
  TEST_NAME="Build etcd-launcher Docker image"
  IMAGE_NAME="quay.io/kubermatic/etcd-launcher:${KUBERMATIC_VERSION}"
  time retry 5 docker build -t "${IMAGE_NAME}" -f cmd/etcd-launcher/Dockerfile .
  time retry 5 kind load docker-image "$IMAGE_NAME" --name ${SEED_NAME}
)

pushElapsed kubermatic_docker_build_duration_milliseconds $beforeDockerBuild

echodate "Successfully built and loaded all images"

# We don't need a valid certificate (and can't even get one), but still need
# to have the CRDs installed so we can at least create a Certificate resource.
TEST_NAME="Deploy cert-manager CRDs"
echodate "Deploying cert-manager CRDs"
retry 5 kubectl apply -f charts/cert-manager/crd/

TEST_NAME="Deploy Kubermatic"
echodate "Deploying Kubermatic using Helm..."

beforeDeployment=$(nowms)

# we always override the quay repositories so we don't have to care if the
# Helm chart is made for CE or EE
retry 3 kubectl create ns kubermatic
retry 3 helm3 --namespace kubermatic install --atomic --timeout 5m \
  --set=kubermatic.domain=ci.kubermatic.io \
  --set=kubermatic.isMaster=true \
  --set=kubermatic.imagePullSecretData="$IMAGE_PULL_SECRET_DATA" \
  --set-string=kubermatic.controller.image.repository="quay.io/kubermatic/kubermatic$REPOSUFFIX" \
  --set-string=kubermatic.masterController.image.repository="quay.io/kubermatic/kubermatic$REPOSUFFIX" \
  --set-string=kubermatic.api.image.repository="quay.io/kubermatic/kubermatic$REPOSUFFIX" \
  --set-string=kubermatic.ui.image.repository="quay.io/kubermatic/dashboard$REPOSUFFIX" \
  --set-string=kubermatic.controller.addons.kubernetes.image.tag="$KUBERMATIC_VERSION" \
  --set-string=kubermatic.controller.image.tag="$KUBERMATIC_VERSION" \
  --set-string=kubermatic.controller.addons.openshift.image.tag="$KUBERMATIC_VERSION" \
  --set-string=kubermatic.api.image.tag="$KUBERMATIC_VERSION" \
  --set=kubermatic.controller.datacenterName="${SEED_NAME}" \
  --set=kubermatic.api.replicas=1 \
  --set-string=kubermatic.masterController.image.tag="$KUBERMATIC_VERSION" \
  --set-string=kubermatic.ui.image.tag=latest \
  --set=kubermatic.ui.replicas=0 \
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
  kubermatic \
  charts/kubermatic/

pushElapsed kubermatic_deployment_duration_milliseconds $beforeDeployment 'method="helm"'

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

appendTrap cleanup_kubermatic_clusters_in_kind EXIT

TEST_NAME="Expose Dex and Kubermatic API"
echodate "Exposing Dex and Kubermatic API to localhost..."
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
