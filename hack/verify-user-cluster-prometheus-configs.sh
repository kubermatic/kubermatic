#!/usr/bin/env bash
set -euo pipefail

echo "This script requires write access to /etc/etcd/pki/client & /etc/kubernetes as it needs to write dummy certs."
echo "This is required because the promtool checks if files inside the prometheus.yaml actually exist."
mkdir -p /etc/etcd/pki/client
touch /etc/etcd/pki/client/apiserver-etcd-client.crt
touch /etc/etcd/pki/client/apiserver-etcd-client.key
mkdir -p /etc/kubernetes
touch /etc/kubernetes/prometheus-client.crt
touch /etc/kubernetes/prometheus-client.key

cd "$(dirname "$0")/../"

for CM in pkg/resources/test/fixtures/configmap-*-prometheus.yaml; do
  echo "Checking ${CM} ..."
  cat ${CM} | yaml2json | jq -r '.data["rules.yaml"]' > rules.yaml
  promtool check rules rules.yaml
  rm rules.yaml

  cat ${CM} | yaml2json | jq -r '.data["prometheus.yaml"]' > prometheus.yaml
  promtool check config prometheus.yaml
  rm prometheus.yaml
done
