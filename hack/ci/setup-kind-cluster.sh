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

source hack/lib.sh

echodate "Setting up kind cluster..."

if [ -z "${JOB_NAME:-}" ] || [ -z "${PROW_JOB_ID:-}" ]; then
  echodate "This script should only be running in a CI environment."
  exit 1
fi

export KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kubermatic}"
export KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ce}"
WITH_WORKERS="${WITH_WORKERS:-}"
WORKERS=''

start_docker_daemon_ci

# Create kind cluster
TEST_NAME="Create kind cluster"

echodate "Preloading the kindest/node image"
docker load --input /kindest.tar

echodate "Creating the kind cluster"
export KUBECONFIG=~/.kube/config

beforeKindCreate=$(nowms)

# If a Docker mirror is available, we tunnel it into the
# kind cluster, which has its own containerd daemon.
# kind current does not allow accessing ports on the host
# from within the cluster and also does not allow adding
# custom flags to the `docker run` call it does in the
# background.
# To circumvent this, we use socat to make the TCP-based
# mirror available as a local socket and then mount this
# into the kind container.
# Since containerd does not support sockets, we also start
# a second socat process in the kind container that unwraps
# the socket again and listens on 127.0.0.1:5001, which is
# then used for containerd.
# Being a docker registry does not incur a lot of requests,
# just a few big ones. For this socat seems pretty reliable.
if [ -n "${DOCKER_REGISTRY_MIRROR_ADDR:-}" ]; then
  mirrorHost="$(echo "$DOCKER_REGISTRY_MIRROR_ADDR" | sed 's#http://##' | sed 's#/+$##g')"

  # make the registry mirror available as a socket,
  # so we can mount it into the kind cluster
  mkdir -p /mirror
  socat UNIX-LISTEN:/mirror/mirror.sock,fork,reuseaddr,unlink-early,mode=777 TCP4:$mirrorHost &

  function end_socat_process {
    echodate "Killing socat docker registry mirror processes..."
    pkill -e socat
  }
  appendTrap end_socat_process EXIT

  if [ -n "${WITH_WORKERS}" ]; then
    WORKERS='  - role: worker
    # mount the socket
    extraMounts:
    - hostPath: /mirror
      containerPath: /mirror
  - role: worker
    # mount the socket
    extraMounts:
    - hostPath: /mirror
      containerPath: /mirror
    '
  fi

  cat << EOF > kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: "${KIND_CLUSTER_NAME}"
nodes:
  - role: control-plane
    # mount the socket
    extraMounts:
    - hostPath: /mirror
      containerPath: /mirror
# we will install Cilium later
networking:
  disableDefaultCNI: true
${WORKERS}
containerdConfigPatches:
  # point to the soon-to-start local socat process
  - |-
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
    endpoint = ["http://127.0.0.1:5001"]
EOF

  kind create cluster --config kind-config.yaml
  pushElapsed kind_cluster_create_duration_milliseconds $beforeKindCreate

  # unwrap the socket inside the kind cluster and make it available on a TCP port,
  # because containerd/Docker doesn't support sockets for mirrors.
  docker exec "$KIND_CLUSTER_NAME-control-plane" bash -c 'apt update --quiet; apt install --quiet socat; socat TCP4-LISTEN:5001,fork,reuseaddr UNIX:/mirror/mirror.sock &'
else
  if [ -n "${WITH_WORKERS}" ]; then
    WORKERS='  - role: worker
  - role: worker'
  fi
  cat << EOF > kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: "${KIND_CLUSTER_NAME}"
# we will install Cilium later
networking:
  disableDefaultCNI: true
nodes:
  - role: control-plane
${WORKERS}
EOF

  kind create cluster --config kind-config.yaml
fi

# This is required if the kindest version matches the user cluster version.
# The kindest image comes with preloaded control plane images, however,
# the preloaded kube-controller-manager image doesn't have cloud providers
# built-in. This is done intentionally in order to reduce the kindest image
# size because kind is used only for local clusters.
# When the kindest version matches the user cluster version, KKP will use the
# preloaded kube-controller-manager image instead of pulling the image from
# k8s.gcr.io. This will cause the kube-controller-manager to crashloop because
# there are no cloud providers in that preloaded image.
# As a solution, we remove the preloaded image after starting the kind
# cluster, which will force KKP to pull the correct image.
docker exec "$KIND_CLUSTER_NAME-control-plane" bash -c "crictl images | grep kube-controller-manager | awk '{print \$2}' | xargs -I{} crictl rmi registry.k8s.io/kube-controller-manager:{}" || true

CILIUM_VERSION="1.18.1"

helm repo add cilium https://helm.cilium.io/
helm install cilium cilium/cilium \
  --version "$CILIUM_VERSION" \
  --namespace kube-system \
  --set image.pullPolicy=IfNotPresent \
  --set ipam.mode=kubernetes \
  --set operator.replicas=1

if [ -z "${DISABLE_CLUSTER_EXPOSER:-}" ]; then
  # Start cluster exposer, which will expose services from within kind as
  # a NodePort service on the host
  echodate "Starting cluster exposer"

  # this is already built and added to the build image, no need to compile it again
  clusterexposer \
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
  KIND_CONTAINER_IP=$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $KIND_CLUSTER_NAME-control-plane)

  iptables -t nat -A PREROUTING -i eth0 -p tcp -m multiport --dports=30000:33000 -j DNAT --to-destination $KIND_CONTAINER_IP
  # By default all traffic gets dropped unless specified (tested with docker server 18.09.1)
  iptables -t filter -I DOCKER-USER -d $KIND_CONTAINER_IP/32 ! -i $KIND_NETWORK_IF -o $KIND_NETWORK_IF -p tcp -m multiport --dports=30000:33000 -j ACCEPT
  # Docker sets up a MASQUERADE rule for postrouting, so nothing to do for us

  echodate "Successfully set up iptables rules for nodeports"
fi

echodate "Kind cluster $KIND_CLUSTER_NAME is up and running."
