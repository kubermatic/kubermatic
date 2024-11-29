#!/usr/bin/env bash

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

### This script builds all the utility container images required by or provided
### in KKP, like the http-prober, S3 exporter, util etc.
### These utilities are not versioned together with KKP and so this script will
### only build and push images if they are not yet present in the target registry.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

export DOCKER_REPO="${DOCKER_REPO:-quay.io/kubermatic}"
export DRY_RUN=${DRY_RUN:-true}

IMAGES="${IMAGES:-}"
if [ -z "$IMAGES" ]; then
  # find all images that have a BUILD_TAG explicitly configured (i.e. are not based on git tags)
  IMAGES="$(find hack/images/ -name settings.sh -exec grep -l BUILD_TAG {} \; | xargs -n1 dirname | xargs -n1 basename | sort)"
fi

if ! $DRY_RUN; then
  echodate "Logging into Quay…"
  retry 5 buildah login -u "$QUAY_IO_USERNAME" -p "$QUAY_IO_PASSWORD" quay.io
fi

image_exists() {
  skopeo inspect "docker://$1" --no-tags > /dev/null 2>&1
}

for image in $IMAGES; do
  TEST_NAME="Build $image image"
  echodate "[$image] Checking…"

  source "hack/images/$image/settings.sh"
  IMAGE_NAME="$DOCKER_REPO/$image:$BUILD_TAG"

  if image_exists "$IMAGE_NAME"; then
    echodate "[$image] $IMAGE_NAME exists already, skipping."
    continue
  fi

  echodate "[$image] Building…"
  hack/images/release.sh "$image"
done

echodate "Done."
