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
### generates a custom CA and a certificate for minio to be used as backup location,
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

cp hack/ci/testdata/kubermatic_backup.yaml $KUBERMATIC_CONFIG

sed -i "s;__IMAGE_PULL_SECRET__;$IMAGE_PULL_SECRET_INLINE;g" $KUBERMATIC_CONFIG
sed -i "s;__KUBERMATIC_DOMAIN__;$KUBERMATIC_DOMAIN;g" $KUBERMATIC_CONFIG

HELM_VALUES_FILE="$(mktemp)"
cat << EOF > $HELM_VALUES_FILE
kubermaticOperator:
  image:
    repository: "quay.io/kubermatic/kubermatic$REPOSUFFIX"
    tag: "$KUBERMATIC_VERSION"

minio:
  storageClass: 'kubermatic-fast'
  certificateSecret: 'minio-tls-cert'
  credentials:
    accessKey: "FXcD7s0tFOPuTv6jaZARJDouc2Hal8E0"
    secretKey: "wdEZGTnhkgBDTDetaHFuizs3pwXHvWTs"
EOF

echodate "Generating custom CA and certificates for minio TLS..."
CUSTOM_CA_KEY=$(mktemp)
CUSTOM_CA_CERT=$(mktemp)
MINIO_TLS_KEY=$(mktemp)
MINIO_TLS_CSR=$(mktemp)
MINIO_TLS_CERT=$(mktemp)

# create CA certificate
openssl genrsa -out "$CUSTOM_CA_KEY" 2048
openssl req -x509 -new -nodes -subj "/C=DE/O=Kubermatic CI/CN=kubermatic-e2e-ca" -key "$CUSTOM_CA_KEY" -sha256 -days 30 -out "$CUSTOM_CA_CERT"

# create private key, CSR and signed certificate for minio TLS
openssl genrsa -out "$MINIO_TLS_KEY" 2048
openssl req -new -nodes \
  -key "$MINIO_TLS_KEY" \
  -subj "/C=DE/O=Kubermatic CI/CN=minio.minio.svc.cluster.local" \
  -config hack/ci/testdata/minio.cnf \
  -out "$MINIO_TLS_CSR"

openssl x509 -req -in "$MINIO_TLS_CSR" \
  -CA "$CUSTOM_CA_CERT" -CAkey "$CUSTOM_CA_KEY" -CAcreateserial \
  -out "$MINIO_TLS_CERT" -days 7 -sha256 \
  -extfile hack/ci/testdata/minio.cnf -extensions req_ext

CA_BUNDLE_CM=$(mktemp)
cat << EOF > $CA_BUNDLE_CM
apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-ca-bundle
  namespace: kubermatic
data:
  ca-bundle.pem: |
$(cat charts/kubermatic-operator/static/ca-bundle.pem $CUSTOM_CA_CERT | sed 's/^/    /')
EOF

kubectl create namespace kubermatic
kubectl create -f $CA_BUNDLE_CM

# prepare CRDs
copy_crds_to_chart
set_crds_version_annotation

# install dependencies and Kubermatic Operator into cluster
TEST_NAME="Install KKP into kind"

./_build/kubermatic-installer deploy --disable-telemetry \
  --storageclass copy-default \
  --config "$KUBERMATIC_CONFIG" \
  --helm-values "$HELM_VALUES_FILE"

# TODO: The installer should wait for everything to finish reconciling.
echodate "Waiting for Kubermatic Operator to deploy Master components..."
# sleep a bit to prevent us from checking the Deployments too early, before
# the operator had time to reconcile
sleep 5
retry 10 check_all_deployments_ready kubermatic

echodate "Finished installing Kubermatic"

echodate "Installing minio..."
kubectl create namespace minio
kubectl create secret tls minio-tls-cert --cert "$MINIO_TLS_CERT" --key "$MINIO_TLS_KEY" --namespace minio
helm --namespace minio upgrade --install --wait --values "$HELM_VALUES_FILE" minio charts/minio/
kubectl apply -f hack/ci/testdata/backup_s3_creds.yaml

echodate "Setting up backup bucket in minio..."
kubectl create -f hack/ci/testdata/minio_bucket_job.yaml
kubectl wait --for=condition=complete --timeout=60s --namespace minio job/create-minio-backup-bucket

TEST_NAME="Setup KKP Seed"
echodate "Installing Seed..."
SEED_MANIFEST="$(mktemp)"
SEED_KUBECONFIG="$(cat $KUBECONFIG | sed 's/127.0.0.1.*/kubernetes.default.svc.cluster.local./' | base64 -w0)"

cp hack/ci/testdata/seed_backup.yaml $SEED_MANIFEST

sed -i "s/__SEED_NAME__/$SEED_NAME/g" $SEED_MANIFEST
sed -i "s/__KUBECONFIG__/$SEED_KUBECONFIG/g" $SEED_MANIFEST

retry 8 kubectl apply --filename $SEED_MANIFEST
retry 8 check_seed_ready kubermatic "$SEED_NAME"
echodate "Finished installing Seed"

sleep 5
echodate "Waiting for Deployments to roll out..."
retry 9 check_all_deployments_ready kubermatic
echodate "Kubermatic is ready."

appendTrap cleanup_kubermatic_clusters_in_kind EXIT
