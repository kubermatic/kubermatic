#!/bin/bash

# See https://kubernetes.io/docs/setup/independent/install-kubeadm/

set -euo pipefail

# Docker
apt-get update
apt-get -y install curl dnsutils iptables ebtables ethtool ca-certificates conntrack util-linux socat jq nfs-common
apt-get install -y docker.io

# Kubernetes
apt-get update && apt-get install -y apt-transport-https
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -

cat <<EOF >/etc/apt/sources.list.d/kubernetes.list
deb http://apt.kubernetes.io/ kubernetes-xenial main
EOF

apt-get update
apt-get install -y kubelet kubeadm kubectl
