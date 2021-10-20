#!/usr/bin/env bash

# Copyright 2021 The Kubermatic Kubernetes Platform contributors.
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

### This script sets up two kind clusters, places a whole lot of KKP
### resources in various states into it and then runs the crd migration
### function of the KKP installer.
### Note that this script does install KKP itself, as KKP needs to be
### completely shutdown during the migration anyway.

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

function ensure_cluster() {
  local name="$1"
  local kubeconfig="$2"

  if kind get clusters | grep -E "^$name\$" >/dev/null; then
    echodate "Cluster $name exists already."
  else
    kind create cluster --kubeconfig "$kubeconfig" --name "$name"
  fi
}

MASTER_CLUSTER_NAME=crdmigration-master
SEED_CLUSTER_NAME=crdmigration-seed

MASTER_KUBECONFIG="$(realpath "$MASTER_CLUSTER_NAME.kubeconfig")"
SEED_KUBECONFIG="$(realpath "$SEED_CLUSTER_NAME.kubeconfig")"

if [ -z "${KEEP_CLUSTERS:-}" ]; then
  echodate "Deleting any previous clusters..."
  kind delete cluster --name "$MASTER_CLUSTER_NAME"
  kind delete cluster --name "$SEED_CLUSTER_NAME"
fi

echodate "Ensuring master and seed clusters (tip: set \$KEEP_CLUSTERS=yes to keep your kind clusters from the previous run)..."
ensure_cluster "$MASTER_CLUSTER_NAME" "$MASTER_KUBECONFIG"
ensure_cluster "$SEED_CLUSTER_NAME" "$SEED_KUBECONFIG"

# note that these must be the _old_ CRDs (*.kubermatic.k8s.io)
echodate "Creating CRDs..."
kubectl --kubeconfig "$MASTER_KUBECONFIG" apply -f charts/kubermatic-operator/crd/
kubectl --kubeconfig "$SEED_KUBECONFIG" apply -f charts/kubermatic-operator/crd/

# To setup proper ownerRefs, we need the UIDs of objects we're
# about to create now; to inject those UIDs we use `sed` and
# work in a temp dir, so that repeated runs of this script can
# work as expected.
# Technically k8s (at least 1.21) doesn't care *what* the UID is,
# but we want to be nice and create proper data here.
WORK_DIR=$(mktemp -d)

function setup_resources() {
  local kubeconfig="$1"
  local sourceDirectory="$2"

  rm -rf -- $WORK_DIR/*.yaml
  cp -ar -- $sourceDirectory/*.yaml "$WORK_DIR"

  export KUBECONFIG="$kubeconfig"
  kubectl apply --filename "$WORK_DIR/00-namespaces.yaml"
  kubectl apply --filename "$WORK_DIR/01-setup.yaml"
  kubectl apply --filename "$WORK_DIR/02-user.yaml"

  # make sure to use fully-qualified kind names, just in case
  # somehow the old and new CRDs are installed already.

  userUID="$(kubectl get users.kubermatic.k8s.io test-user -o jsonpath='{.metadata.uid}')"
  sed -i "s/__USER_UID__/$userUID/g" $WORK_DIR/*

  kubectl apply --filename "$WORK_DIR/03-project.yaml"

  projectUID="$(kubectl get projects.kubermatic.k8s.io kkpproject -o jsonpath='{.metadata.uid}')"
  sed -i "s/__PROJECT_UID__/$projectUID/g" $WORK_DIR/*

  kubectl apply --filename "$WORK_DIR/04-project-contents.yaml"
  kubectl apply --filename "$WORK_DIR/05-cluster.yaml"

  clusterUID="$(kubectl get clusters.kubermatic.k8s.io kkpcluster -o jsonpath='{.metadata.uid}')"
  sed -i "s/__CLUSTER_UID__/$clusterUID/g" $WORK_DIR/*

  kubectl apply --filename "$WORK_DIR/06-cluster-contents.yaml"
  kubectl apply --filename "$WORK_DIR/07-misc.yaml"
}

echodate "Creating resources in master cluster..."
setup_resources "$MASTER_KUBECONFIG" hack/ci/testdata/crdmigration/master

# create kubeconfig Secret for the seed kind cluster
kubectl \
  --kubeconfig="$MASTER_KUBECONFIG" \
  --namespace=kubermatic \
  create secret generic kubeconfig-crdmigration \
    --from-file="kubeconfig=$SEED_KUBECONFIG" \
    --dry-run=client \
    --output=yaml |
    kubectl apply --filename -

echodate "Creating resources in seed cluster..."
setup_resources "$SEED_KUBECONFIG" hack/ci/testdata/crdmigration/seed

export KUBERMATIC_EDITION=ee
make clean kubermatic-installer

export KUBECONFIG="$MASTER_KUBECONFIG"
_build/kubermatic-installer migrate-crds
