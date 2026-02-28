# Copyright 2024 The Kubermatic Kubernetes Platform contributors.
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

if [ -z "${KIND_CLUSTER_NAME:-}" ]; then
  echodate "KIND_CLUSTER_NAME must be set by calling setup-kind-cluster.sh first."
  exit 1
fi

# The Kubermatic version to build.
export KUBERMATIC_VERSION="${KUBERMATIC_VERSION:-$(git rev-parse HEAD)}"

REPOSUFFIX=""
if [ "$KUBERMATIC_EDITION" != "ce" ]; then
  REPOSUFFIX="-$KUBERMATIC_EDITION"
fi

# This is just used as a const
export SEED_NAME=kubermatic

# Set docker config
echo "$IMAGE_PULL_SECRET_DATA" | base64 -d > /config.json

# Build binaries and load the Docker images into the kind cluster
echodate "Building binaries for $KUBERMATIC_VERSION"
TEST_NAME="Build Kubermatic binaries"

beforeGoBuild=$(nowms)
time retry 1 make build
pushElapsed kubermatic_go_build_duration_milliseconds $beforeGoBuild

beforeDockerBuild=$(nowms)

(
  echodate "Building Kubermatic Docker image"
  TEST_NAME="Build Kubermatic Docker image"
  IMAGE_NAME="quay.io/kubermatic/kubermatic$REPOSUFFIX:$KUBERMATIC_VERSION"
  time retry 5 docker build -t "$IMAGE_NAME" .
  time retry 5 kind load docker-image "$IMAGE_NAME" --name "$KIND_CLUSTER_NAME"
)
(
  echodate "Building addons image"
  TEST_NAME="Build addons Docker image"
  cd addons
  IMAGE_NAME="quay.io/kubermatic/addons:$KUBERMATIC_VERSION"
  time retry 5 docker build -t "${IMAGE_NAME}" .
  time retry 5 kind load docker-image "$IMAGE_NAME" --name "$KIND_CLUSTER_NAME"
)
(
  echodate "Building nodeport-proxy image"
  TEST_NAME="Build nodeport-proxy Docker image"
  cd cmd/nodeport-proxy
  make build
  IMAGE_NAME="quay.io/kubermatic/nodeport-proxy:$KUBERMATIC_VERSION"
  time retry 5 docker build -t "${IMAGE_NAME}" .
  time retry 5 kind load docker-image "$IMAGE_NAME" --name "$KIND_CLUSTER_NAME"
)
(
  echodate "Building etcd-launcher image"
  TEST_NAME="Build etcd-launcher Docker image"
  IMAGE_NAME="quay.io/kubermatic/etcd-launcher:${KUBERMATIC_VERSION}"
  time retry 5 docker build -t "${IMAGE_NAME}" -f cmd/etcd-launcher/Dockerfile .
  time retry 5 kind load docker-image "$IMAGE_NAME" --name "$KIND_CLUSTER_NAME"
)
(
  echodate "Building network-interface-manager image"
  TEST_NAME="Build network-interface-manager Docker image"
  cd cmd/network-interface-manager
  make build
  IMAGE_NAME="quay.io/kubermatic/network-interface-manager:$KUBERMATIC_VERSION"
  time retry 5 docker build -t "${IMAGE_NAME}" .
  time retry 5 kind load docker-image "$IMAGE_NAME" --name "$KIND_CLUSTER_NAME"
)

pushElapsed kubermatic_docker_build_duration_milliseconds $beforeDockerBuild
echodate "Successfully built and loaded all images"

# prepare to run kubermatic-installer
KUBERMATIC_CONFIG="$(mktemp)"
IMAGE_PULL_SECRET_INLINE="$(echo "$IMAGE_PULL_SECRET_DATA" | base64 --decode | jq --compact-output --monochrome-output '.')"
KUBERMATIC_DOMAIN="${KUBERMATIC_DOMAIN:-ci.kubermatic.io}"

cp hack/ci/testdata/kubermatic_cluster_backup.yaml $KUBERMATIC_CONFIG

sed -i "s;__SERVICE_ACCOUNT_KEY__;$SERVICE_ACCOUNT_KEY;g" $KUBERMATIC_CONFIG
sed -i "s;__IMAGE_PULL_SECRET__;$IMAGE_PULL_SECRET_INLINE;g" $KUBERMATIC_CONFIG
sed -i "s;__KUBERMATIC_DOMAIN__;$KUBERMATIC_DOMAIN;g" $KUBERMATIC_CONFIG

HELM_VALUES_FILE="$(mktemp)"
cat << EOF > $HELM_VALUES_FILE
kubermaticOperator:
  image:
    repository: "quay.io/kubermatic/kubermatic$REPOSUFFIX"
    tag: "$KUBERMATIC_VERSION"
EOF

# prepare CRDs
copy_crds_to_chart
set_crds_version_annotation

# install dependencies and Kubermatic Operator into cluster
TEST_NAME="Install KKP into kind"

echodate "Installing Master..."
./_build/kubermatic-installer deploy \
  --disable-telemetry \
  --storageclass copy-default \
  --config "$KUBERMATIC_CONFIG" \
  --helm-values "$HELM_VALUES_FILE"

# TODO: The installer should wait for everything to finish reconciling.
echodate "Waiting for Kubermatic Operator to deploy Master components..."
# sleep a bit to prevent us from checking the Deployments too early, before
# the operator had time to reconcile
sleep 5
retry 10 check_all_deployments_ready kubermatic

TEST_NAME="Setup KKP Seed"
echodate "Installing Seed..."
SEED_MANIFEST="$(mktemp)"
SEED_KUBECONFIG="$(cat $KUBECONFIG | sed 's/127.0.0.1.*/kubernetes.default.svc.cluster.local./' | base64 -w0)"

cp hack/ci/testdata/seed.yaml $SEED_MANIFEST

sed -i "s/__SEED_NAME__/$SEED_NAME/g" $SEED_MANIFEST
sed -i "s/__KUBECONFIG__/$SEED_KUBECONFIG/g" $SEED_MANIFEST

retry 8 kubectl apply --filename $SEED_MANIFEST
retry 8 check_seed_ready kubermatic "$SEED_NAME"
echodate "Finished installing Seed"

sleep 5
echodate "Waiting for Deployments to roll out..."
retry 9 check_all_deployments_ready kubermatic

appendTrap cleanup_kubermatic_clusters_in_kind EXIT

echodate "Finished installing Kubermatic."
