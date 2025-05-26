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

set -euo pipefail
# Required for signal propagation to work so
# the cleanup trap gets executed when the script
# receives a SIGINT
set -o monitor

cd $(dirname $0)/..
source hack/lib.sh

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache..."

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

export CGO_ENABLED=1

LOCALSTACK_TAG="${LOCALSTACK_TAG:-3.0.1}"
LOCALSTACK_IMAGE="${LOCALSTACK_IMAGE:-localstack/localstack:$LOCALSTACK_TAG}"

if [[ ! -z "${JOB_NAME:-}" ]] && [[ ! -z "${PROW_JOB_ID:-}" ]]; then
  start_docker_daemon_ci
fi

if ! [ -x "$(command -v etcd)" ]; then
  TEST_NAME="Download envtest binaries"
  echodate "Downloading envtest binaries..."

  TMP_DIR="$(mktemp -d)"
  PATH="$PATH:$TMP_DIR"

  download_envtest "$TMP_DIR" "1.33.0"
fi

# For the AWS tests, we need a localstack container running.
if [ -z "${SKIP_AWS_PROVIDER:-}" ]; then
  echodate "Setting up localstack container, set \$SKIP_AWS_PROVIDER to skip..."

  containerName=kkp-localstack

  docker run \
    --name "$containerName" \
    --rm \
    --detach \
    --publish 4566:4566 \
    --publish 4571:4571 \
    --env "SERVICES=iam,ec2" \
    "$LOCALSTACK_IMAGE"

  function stop_localstack() {
    echodate "Stopping localstack container..."
    docker stop "$containerName"
  }
  trap stop_localstack EXIT SIGINT SIGTERM

  export AWS_ACCESS_KEY_ID=test
  export AWS_SECRET_ACCESS_KEY=test
  export AWS_REGION=eu-north-1

  # the existence of this env var enables the AWS provider's integration tests
  export AWS_TEST_ENDPOINT=http://localhost:4566
fi

# For the kubectl tests, we must build the final KKP docker image.
if [ -z "${SKIP_KUBECTL_TESTS:-}" ]; then
  echodate "Building dummy KKP image, set \$SKIP_KUBECTL_TESTS to skip..."

  # we do not need actual KKP binaries in the image
  mkdir _build
  touch \
    _build/kubermatic-operator \
    _build/kubermatic-installer \
    _build/kubermatic-webhook \
    _build/master-controller-manager \
    _build/seed-controller-manager \
    _build/user-cluster-controller-manager \
    _build/user-cluster-webhook

  # the existence of this env var enables the integration tests
  export KUBECTL_TEST_IMAGE=kkpkubectltest

  docker build -t $KUBECTL_TEST_IMAGE .
fi

echodate "Running integration tests..."

# Run integration tests and only integration tests by:
# * Finding all files that contain the build tag via grep
# * Extracting the dirname as the `go test` command doesn't play well with individual files as args
# * Prefixing them with `./` as that's needed by `go test` as well
for file in $(grep --files-with-matches --recursive --extended-regexp '//go:build.+integration' cmd/ pkg/ | xargs dirname | sort -u); do
  echodate "Testing package ${file}..."

  if [[ "$file" =~ .*vsphere.* ]] && provider_disabled vsphere; then
    continue
  fi

  extraArgs=()
  if [[ "$file" =~ .*helmclient.* ]] || [[ "$file" =~ .*providers/template.* ]]; then
    extraArgs+=(-registry-hostname "$HOSTNAME")
  fi

  go_test $(echo $file | sed 's/\//_/g') -tags "integration,${KUBERMATIC_EDITION:-ce}" -race ./${file} -v "${extraArgs[@]}"
done
