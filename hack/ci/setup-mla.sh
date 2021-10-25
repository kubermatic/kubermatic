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
git clone "$URL" "$tempdir"
(
  cd "$tempdir"
  helm --namespace mla upgrade --atomic --create-namespace --install mla-secrets charts/mla-secrets --values config/mla-secrets/values.yaml
  ./hack/deploy-seed.sh
)

sleep 5
echodate "Waiting for MLA to deploy Seed components..."
retry 8 check_all_deployments_ready mla

TEST_NAME="Expose Grafana"
echodate "Exposing Grafana to localhost..."
kubectl port-forward --address 0.0.0.0 -n mla svc/grafana 3000:80 > /dev/null &
echodate "Finished exposing components"

echodate "MLA is ready."
