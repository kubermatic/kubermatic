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
# NB: The CE requires Seeds to be named this way
export SEED_NAME=kubermatic

# This defines the Kubermatic API endpoint the e2e tests will communicate with.
# The api service is kubectl-proxied later on.
export KUBERMATIC_API_ENDPOINT="http://localhost:8080"

# Tell the conformance tester what dummy account we configure for the e2e tests.
export KUBERMATIC_DEX_VALUES_FILE=$(realpath hack/ci/testdata/oauth_values.yaml)
export KUBERMATIC_OIDC_LOGIN="roxy@loodse.com"
export KUBERMATIC_OIDC_PASSWORD="password"

# Set docker config
echo "$IMAGE_PULL_SECRET_DATA" | base64 -d > /config.json

# The alias makes it easier to access the port-forwarded Dex inside the Kind cluster;
# the token issuer cannot be localhost:5556, because pods inside the cluster would not
# find Dex anymore. As this script can be run multiple times in the same CI job,
# we must make sure to only add the alias once.
if ! grep oauth /etc/hosts > /dev/null; then
  echodate "Setting dex.oauth alias in /etc/hosts"
  # The container runtime allows us to change the content but not to change the inode
  # which is what sed -i does, so write to a tempfile and write the tempfiles content back.
  temp_hosts="$(mktemp)"
  sed 's/localhost/localhost dex.oauth/' /etc/hosts > $temp_hosts
  cat $temp_hosts > /etc/hosts
  echodate "Set dex.oauth alias in /etc/hosts"
fi

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
  echodate "Building kubeletdnat-controller image"
  TEST_NAME="Build kubeletdnat-controller Docker image"
  cd cmd/kubeletdnat-controller
  make build
  IMAGE_NAME="quay.io/kubermatic/kubeletdnat-controller:$KUBERMATIC_VERSION"
  time retry 5 docker build -t "${IMAGE_NAME}" .
  time retry 5 kind load docker-image "$IMAGE_NAME" --name "$KIND_CLUSTER_NAME"
)
(
  echodate "Building user-ssh-keys-agent image"
  TEST_NAME="Build user-ssh-keys-agent Docker image"
  retry 5 docker login -u "$QUAY_IO_USERNAME" -p "$QUAY_IO_PASSWORD" quay.io
  IMAGE_NAME=quay.io/kubermatic/user-ssh-keys-agent:$KUBERMATIC_VERSION
  time retry 5 docker build -f cmd/user-ssh-keys-agent/Dockerfile -t "${IMAGE_NAME}" .
  time retry 5 docker push "${IMAGE_NAME}"
)
(
  echodate "Building etcd-launcher image"
  TEST_NAME="Build etcd-launcher Docker image"
  IMAGE_NAME="quay.io/kubermatic/etcd-launcher:${KUBERMATIC_VERSION}"
  time retry 5 docker build -t "${IMAGE_NAME}" -f cmd/etcd-launcher/Dockerfile .
  time retry 5 kind load docker-image "$IMAGE_NAME" --name "$KIND_CLUSTER_NAME"
)

pushElapsed kubermatic_docker_build_duration_milliseconds $beforeDockerBuild
echodate "Successfully built and loaded all images"

# prepare to run kubermatic-installer
KUBERMATIC_CONFIG="$(mktemp)"
IMAGE_PULL_SECRET_INLINE="$(echo "$IMAGE_PULL_SECRET_DATA" | base64 --decode | jq --compact-output --monochrome-output '.')"
KUBERMATIC_DOMAIN="${KUBERMATIC_DOMAIN:-ci.kubermatic.io}"

cp hack/ci/testdata/kubermatic.yaml $KUBERMATIC_CONFIG

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

# append custom Dex configuration
cat hack/ci/testdata/oauth_values.yaml >> $HELM_VALUES_FILE

# install dependencies and Kubermatic Operator into cluster
./_build/kubermatic-installer deploy \
  --storageclass copy-default \
  --config "$KUBERMATIC_CONFIG" \
  --helm-values "$HELM_VALUES_FILE" \
  --helm-binary "helm3"

# TODO: The installer should wait for everything to finish reconciling.
echodate "Waiting for Kubermatic Operator to deploy Master components..."
# sleep a bit to prevent us from checking the Deployments too early, before
# the operator had time to reconcile
sleep 5
retry 10 check_all_deployments_ready kubermatic

echodate "Finished installing Kubermatic"

echodate "Installing Seed..."
SEED_MANIFEST="$(mktemp)"
SEED_KUBECONFIG="$(cat $KUBECONFIG | sed 's/127.0.0.1.*/kubernetes.default.svc.cluster.local./' | base64 -w0)"

cp hack/ci/testdata/seed.yaml $SEED_MANIFEST

sed -i "s/__SEED_NAME__/$SEED_NAME/g" $SEED_MANIFEST
sed -i "s/__BUILD_ID__/$BUILD_ID/g" $SEED_MANIFEST
sed -i "s/__KUBECONFIG__/$SEED_KUBECONFIG/g" $SEED_MANIFEST

retry 8 kubectl apply -f $SEED_MANIFEST
echodate "Finished installing Seed"

sleep 5
echodate "Waiting for Kubermatic Operator to deploy Seed components..."
retry 8 check_all_deployments_ready kubermatic
echodate "Kubermatic Seed is ready."

echodate "Waiting for VPA to be ready..."
retry 8 check_all_deployments_ready kube-system
echodate "VPA is ready."

appendTrap cleanup_kubermatic_clusters_in_kind EXIT

TEST_NAME="Expose Dex and Kubermatic API"
echodate "Exposing Dex and Kubermatic API to localhost..."
kubectl port-forward --address 0.0.0.0 -n oauth svc/dex 5556 > /dev/null &
kubectl port-forward --address 0.0.0.0 -n kubermatic svc/kubermatic-api 8080:80 > /dev/null &
echodate "Finished exposing components"
