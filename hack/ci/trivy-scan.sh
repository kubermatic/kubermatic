#!/usr/bin/env bash

# Copyright 2023 The Kubermatic Kubernetes Platform contributors.
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

cd $(dirname $0)/../..
source hack/lib.sh

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

export DOCKER_REPO=quay.io/kubermatic
export TAGS=local          # use a fixed tag
export ARCHITECTURES=amd64 # save time by skipping building/scanning arm64
export UIDOCKERTAG=NA      # prevent the need for an SSH key to git ls-remote the dashboard version

TEST_NAME="Create container images"
echodate "Creating container images"
NO_PUSH=true hack/ci/release-images.sh

echodate "Images built successfully."

REPOSUFFIX=""
if [ "$KUBERMATIC_EDITION" != "ce" ]; then
  REPOSUFFIX="-$KUBERMATIC_EDITION"
fi

images=(
  "$DOCKER_REPO/kubermatic$REPOSUFFIX:$TAGS"
  "$DOCKER_REPO/nodeport-proxy:$TAGS"
  "$DOCKER_REPO/etcd-launcher:$TAGS"
  "$DOCKER_REPO/addons:$TAGS"
  "$DOCKER_REPO/user-ssh-keys-agent:$TAGS-amd64"
  "$DOCKER_REPO/network-interface-manager:$TAGS-amd64"
)

TRIVY=aquasec/trivy:0.56.2

mkdir -p /tmp/trivy-cache

echodate "Scanning Go dependencies"
time docker run \
  --rm \
  -v "$(realpath .):/go/src/k8c.io/kubermatic" \
  -v /run/docker.sock:/var/run/docker.sock \
  -v /tmp/trivy-cache:/cache \
  -e "TRIVY_DB_REPOSITORY=public.ecr.aws/aquasecurity/trivy-db" \
  -e "TRIVY_JAVA_DB_REPOSITORY=public.ecr.aws/aquasecurity/trivy-java-db" \
  -w /go/src/k8c.io/kubermatic \
  $TRIVY --cache-dir /cache fs --quiet .

# improve readability from far away
echo
echo
echo

for image in "${images[@]}"; do
  TEST_NAME="Scan $image"
  echodate "Scanning $image"

  time docker run \
    --rm \
    -v "$(realpath .):/opt/src" \
    -v /run/docker.sock:/var/run/docker.sock \
    -v /tmp/trivy-cache:/cache \
    -e "TRIVY_DB_REPOSITORY=public.ecr.aws/aquasecurity/trivy-db" \
    -e "TRIVY_JAVA_DB_REPOSITORY=public.ecr.aws/aquasecurity/trivy-java-db" \
    -w /opt/src \
    $TRIVY --cache-dir /cache image --quiet "$image"

  # improve readability from far away
  echo
  echo
  echo
done

echodate "Scans completed."
