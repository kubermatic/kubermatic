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

###
### This script is be used to build all of the container images that are part
### of KKP. This includes both the images that are versioned and released for
### each KKP release (like the main image or the s3-exporter), and the images
### that are versioned independently (like the util image).
###
### The script does some caching magic in order to speed up building multiple
### images in CI. This is triggered by setting the $GOCACHE environment variable.
### Without a Go cache, no caching magic is applied (and the builds will take
### much longer).
###
### Since Docker as of November 2024 still has issues building multiarch images
### in multiple steps (if buildx is used, the `docker buildx build` commands will
### not be able to access the temporary gocache images, unless we push those to
### a real registry), Buildah is used to produce the images. With Buildah it just
### works(tm) without any hassle.
### However developers can still freely use Docker or other tools to build the
### images directly, ignoring this script entirely and for example just doing
###
###   $ docker build -t addons -f hack/images/addons/Dockerfile .
###

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

IMAGES_DIR=hack/images
KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ce}"
DOCKER_REPO="${DOCKER_REPO:-quay.io/kubermatic}"
DRY_RUN=${DRY_RUN:-true}
ARCHITECTURES="${ARCHITECTURES:-amd64 arm64}"

IMAGES_TO_BUILD="$@"
if [ -z "$IMAGES_TO_BUILD" ]; then
  echo "Usage: ./release.sh image1 image2 image3"
  exit 1
fi

REPOSUFFIX=""
if [ "$KUBERMATIC_EDITION" != "ce" ]; then
  REPOSUFFIX="-$KUBERMATIC_EDITION"
fi

# When building many container images, copying the gocache into the docker build
# context gets expensive really fast (each multiarch image needs to copy ~4GB of
# data for each architecture), so to reduce this, we make use of "caching". Most
# of the caching solutions provided by Docker would either not allow to prime
# the cache from a .tar file like we do, or require to have the caching data be
# part of the build context, which is exactly what we try to avoid).
# By creating a custom base image that is just a golang image with our cache
# baked in, all of our Dockerfiles can be use this temporary base image as their
# base.

CUSTOM_BASE_IMAGE=

if [ -n "${GOCACHE_BASE:-}" ]; then
  echodate "Building custom base images…"

  CUSTOM_BASE_IMAGE="kkpcache:local"

  for ARCH in $ARCHITECTURES; do
    FULL_NAME="$CUSTOM_BASE_IMAGE-$ARCH"

    if buildah images --quiet "$FULL_NAME" 1>/dev/null 2>&1; then
      echodate "Cache image $FULL_NAME exists already, skipping."
    else
      echodate "Building $CUSTOM_BASE_IMAGE-$ARCH…"
      buildah build \
        --platform "linux/$ARCH" \
        --file "$IMAGES_DIR/base/Dockerfile" \
        --tag "$FULL_NAME" \
        "$GOCACHE_BASE/$ARCH"
    fi
  done
else
  echodate "No \$GOCACHE_BASE defined, skipping custom base images."
fi

# Cache is ready, now we can proceed to build each requested image individually.
ALL_MANIFESTS=

for REPOSITORY in $IMAGES_TO_BUILD; do
  echodate "Building $REPOSITORY image…"

  unset BUILD_TAG
  unset BUILD_CONTEXT
  unset BUILD_PER_EDITION

  # load image settings; this sets $BUILD_TAG (optional), $BUILD_CONTEXT
  source "$IMAGES_DIR/$REPOSITORY/settings.sh"

  FULL_REPOSITORY="$REPOSITORY"
  if [[ -n "${BUILD_PER_EDITION:-}" ]]; then
    FULL_REPOSITORY+="$REPOSUFFIX"
  fi

  # a BUILD_TAG has precedence over a $TAG environment variable
  IMAGE_TAG="${BUILD_TAG:-}"

  if [ -z "$IMAGE_TAG" ]; then
    IMAGE_TAG="${TAG:-}"

    if [ -z "$IMAGE_TAG" ]; then
      echo "Error: no \$TAG defined in environment and no \$BUILD_TAG in $REPOSITORY/settings.sh."
      exit 1
    fi
  fi

  MANIFEST_NAME="$DOCKER_REPO/$FULL_REPOSITORY:$IMAGE_TAG"
  ALL_MANIFESTS="$ALL_MANIFESTS $MANIFEST_NAME"

  # Images within a manifest are not unique per architecture, so if this script
  # is run multiple times, the images would just pile up in the manifest.
  if buildah manifest exists "$MANIFEST_NAME"; then
    buildah manifest rm "$MANIFEST_NAME"
  fi

  buildah manifest create "$MANIFEST_NAME"

  # build it for each architecture
  for ARCH in $ARCHITECTURES; do
    IMAGE="$MANIFEST_NAME-$ARCH"
    BASE_IMAGE="$CUSTOM_BASE_IMAGE"

    if [ -n "$BASE_IMAGE" ]; then
      BASE_IMAGE="$BASE_IMAGE-$ARCH"
    fi

    echodate "Building $IMAGE…"

    args=()
    [ -n "${KUBERMATICCOMMIT:-}" ]    && args+=(--env "KUBERMATICCOMMIT=$KUBERMATICCOMMIT")
    [ -n "${KUBERMATICDOCKERTAG:-}" ] && args+=(--env "KUBERMATICDOCKERTAG=$KUBERMATICDOCKERTAG")
    [ -n "${UIDOCKERTAG:-}" ]         && args+=(--env "UIDOCKERTAG=$UIDOCKERTAG")
    [ -n "${BASE_IMAGE:-}" ]          && args+=(--build-arg "BASE_IMAGE=$BASE_IMAGE")
    [ -n "${GOPROXY:-}" ]             && args+=(--build-arg "GOPROXY=$GOPROXY")
    [ -n "${KUBERMATIC_EDITION:-}" ]  && args+=(--build-arg "KUBERMATIC_EDITION=$KUBERMATIC_EDITION")

    (
      set -x
      buildah build \
        --platform "linux/$ARCH" \
        --manifest "$MANIFEST_NAME" \
        --file "$IMAGES_DIR/$REPOSITORY/Dockerfile" \
        --tag "$IMAGE" \
        --label "org.opencontainers.image.version=$IMAGE_TAG" \
        ${args[@]} \
        ${BUILD_CONTEXT:-.}
    )
  done
done

if $DRY_RUN; then
  echodate "Set \$DRY_RUN=false to push the image(s) you just built."
else
  for MANIFEST in $ALL_MANIFESTS; do
    echodate "Publishing $MANIFEST…"
    buildah manifest push --all "$MANIFEST" "docker://$MANIFEST"
  done

  echodate "Done publishing images."
fi
