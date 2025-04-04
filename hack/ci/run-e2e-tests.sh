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

### This script sets up a local KKP installation in kind and then
### runs the conformance-tester to create userclusters and check their
### Kubernetes conformance.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

TEST_NAME="Download kube-test binaries"
echodate "Downloading kube-test binaries..."

echodate "Kubernetes release: $RELEASES_TO_TEST"

KUBE_VERSION="$(download_archive https://dl.k8s.io/release/stable-$RELEASES_TO_TEST.txt -Ls)"
echodate "Kubernetes version: $KUBE_VERSION"

TMP_DIR="/tmp/k8s"
DIRECTORY="/opt/kube-test/$RELEASES_TO_TEST"

rm -rf -- "$DIRECTORY"
mkdir -p "$TMP_DIR" "$DIRECTORY"

DIRECTORY="$DIRECTORY/platforms/$(go env GOOS)/$(go env GOARCH)"
mkdir -p "$DIRECTORY"

TEST_BINARIES=kubernetes-test-$(go env GOOS)-$(go env GOARCH).tar.gz
download_archive "https://dl.k8s.io/$KUBE_VERSION/$TEST_BINARIES" -Lo "$TMP_DIR/$TEST_BINARIES"
tar -zxf "$TMP_DIR/$TEST_BINARIES" -C "$TMP_DIR"
mv $TMP_DIR/kubernetes/test/bin/* "$DIRECTORY/"
rm -rf -- "$TMP_DIR"

echodate "Done downloading Kubernetes test binaries."

export GIT_HEAD_HASH="$(git rev-parse HEAD)"
export KUBERMATIC_VERSION="${GIT_HEAD_HASH}"

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

echodate "Creating kind cluster"
export KIND_CLUSTER_NAME="${SEED_NAME:-kubermatic}"
source hack/ci/setup-kind-cluster.sh

# gather the logs of all things in the cluster control plane and in the Kubermatic namespace
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/cluster-control-plane" --namespace 'cluster-*' > /dev/null 2>&1 &
protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &

echodate "Setting up Kubermatic in kind on revision ${KUBERMATIC_VERSION}"

export KUBERMATIC_YAML=hack/ci/testdata/kubermatic.yaml

beforeKubermaticSetup=$(nowms)
source hack/ci/setup-kubermatic-in-kind.sh
pushElapsed kind_kubermatic_setup_duration_milliseconds $beforeKubermaticSetup

echodate "Running conformance tests"
./hack/ci/run-conformance-tests.sh
