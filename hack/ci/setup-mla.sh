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

### This script creates a local kind cluster, compiles the KKP binaries,
### creates all Docker images and loads them into the kind cluster,
### then installs KKP using the KKP installer + operator and sets up a
### single shared master/seed system.
### This serves as the precursor for all other tests.
###
### This script should be sourced, not called, so callers get the variables
### it sets.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

URL="https://github.com/kubermatic/mla.git"

# Ensure Github's host key is available and disable IP checking.
# ensure_github_host_pubkey

echodate "Cloning MLA repo"
# clone the target and pick the right branch
tempdir="$(mktemp -d)"
trap "rm -rf '$tempdir'" EXIT
# We intentionally force an existing release here
git clone --depth 1 --branch release/v0.1 "$URL" "$tempdir"
(
  cd "$tempdir"

  # due to the anti affinities, getting more than 1 replica can take forever in kind,
  # so we reduce the replica count for all those components
  yq write --inplace config/cortex/values.yaml cortex.ingester.replicas 1
  yq write --inplace config/cortex/values.yaml cortex.distributor.replicas 1
  yq write --inplace config/cortex/values.yaml cortex.alertmanager.replicas 1

  yq write --inplace config/loki/values.yaml loki-distributed.ingester.replicas 1
  yq write --inplace config/loki/values.yaml loki-distributed.distributor.replicas 1

  # ensure that dependencies are fetched and stored in the corresponding charts dir
  echodate "Fetching dependencies for the charts"
  hack/fetch-chart-dependencies.sh

  helm --namespace mla upgrade --atomic --create-namespace --install mla-secrets charts/mla-secrets --values config/mla-secrets/values.yaml
  helm --namespace mla upgrade --atomic --create-namespace --install minio charts/minio --values config/minio/values.yaml
  helm --namespace mla upgrade --atomic --create-namespace --install grafana charts/grafana --values config/grafana/values.yaml
  kubectl apply -f dashboards/
  kubectl create -n mla configmap cortex-runtime-config --from-file=config/cortex/runtime-config.yaml || true
  helm --namespace mla upgrade --atomic --create-namespace --install consul charts/consul --values config/consul/values.yaml
  helm --namespace mla upgrade --atomic --create-namespace --install --debug cortex charts/cortex --values config/cortex/values.yaml --timeout 20m
  helm --namespace mla upgrade --atomic --create-namespace --install loki-distributed charts/loki-distributed --values config/loki/values.yaml --timeout 10m
  helm --namespace mla upgrade --atomic --create-namespace --install alertmanager-proxy charts/alertmanager-proxy
  helm --namespace mla upgrade --atomic --create-namespace --install minio-lifecycle-mgr charts/minio-lifecycle-mgr --values config/minio-lifecycle-mgr/values.yaml
  ./hack/deploy-seed.sh
)

sleep 5
echodate "Waiting for MLA to deploy Seed components..."
retry 8 check_all_deployments_ready mla

TEST_NAME="Expose Grafana"
echodate "Exposing Grafana to localhost..."
kubectl port-forward --address 0.0.0.0 -n mla svc/grafana 3000:80 > /dev/null &
kubectl port-forward --address 0.0.0.0 -n mla svc/cortex-alertmanager 3001:8080 > /dev/null &
kubectl port-forward --address 0.0.0.0 -n mla svc/cortex-ruler 3002:8080 > /dev/null &
kubectl port-forward --address 0.0.0.0 -n mla svc/loki-distributed-ruler 3003:3100 > /dev/null &
echodate "Finished exposing components"

echodate "MLA is ready."
