#!/usr/bin/env bash

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

### This ensures that the Prometheus rules deployed into userclusters
### are valid Prometheus rules.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

echodate "This script requires write access to /etc/etcd/pki/client & /etc/kubernetes as it needs to write dummy certs."
echodate "This is required because the promtool checks if files inside the prometheus.yaml actually exist."

mkdir -p /etc/etcd/pki/client
touch /etc/etcd/pki/client/apiserver-etcd-client.crt
touch /etc/etcd/pki/client/apiserver-etcd-client.key
mkdir -p /etc/kubernetes
touch /etc/kubernetes/prometheus-client.crt
touch /etc/kubernetes/prometheus-client.key

for CM in pkg/resources/test/fixtures/configmap-*-prometheus.yaml; do
  echodate "Checking ${CM} ..."
  cat ${CM} | yaml2json | jq -r '.data["rules.yaml"]' > rules.yaml
  promtool check rules rules.yaml
  rm rules.yaml

  cat ${CM} | yaml2json | jq -r '.data["prometheus.yaml"]' > prometheus.yaml
  promtool check config prometheus.yaml
  rm prometheus.yaml
done
