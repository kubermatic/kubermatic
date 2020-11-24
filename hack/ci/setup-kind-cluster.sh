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

start_docker_daemon

# Prevent mtu-related timeouts
echodate "Setting iptables rule to clamp mss to path mtu"
iptables -t mangle -A POSTROUTING -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu

# Make debugging a bit better
echodate "Configuring bash"
cat <<EOF >>~/.bashrc
# Gets set to the CI clusters kubeconfig from a preset
unset KUBECONFIG

cn() {
  kubectl config set-context --current --namespace=\$1
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

# Load kind image
echodate "Loading kindest image"
docker load --input /kindest.tar
echodate "Loaded kindest image"

# Create kind cluster
TEST_NAME="Create kind cluster"
echodate "Creating the kind cluster"
export KUBECONFIG=~/.kube/config

beforeKindCreate=$(nowms)
export KIND_NODE_VERSION=v1.18.2
kind create cluster --name "$KIND_CLUSTER_NAME" --image=kindest/node:$KIND_NODE_VERSION
pushElapsed kind_cluster_create_duration_milliseconds $beforeKindCreate "node_version=\"$KIND_NODE_VERSION\""

# Start cluster exposer, which will expose services from within kind as
# a NodePort service on the host
echodate "Starting cluster exposer"

CGO_ENABLED=0 go build --tags "$KUBERMATIC_EDITION" -v -o /tmp/clusterexposer ./pkg/test/clusterexposer/cmd
/tmp/clusterexposer \
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
# By default all traffic gets dropped unless specified (tested with docker server 18.09.1)
iptables -t filter -I DOCKER-USER -d 172.18.0.2/32 ! -i $KIND_NETWORK_IF -o $KIND_NETWORK_IF -p tcp -m multiport --dports=30000:33000 -j ACCEPT
# Docker sets up a MASQUERADE rule for postrouting, so nothing to do for us

echodate "Successfully set up iptables rules for nodeports"
echodate "Kind cluster $KIND_CLUSTER_NAME using Kubernetes $KIND_NODE_VERSION is up and running."
