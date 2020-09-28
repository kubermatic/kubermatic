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

set -euo pipefail

# Required for signal propagation to work so
# the cleanup trap gets executed when the script
# receives a SIGINT
set -o monitor

# The gocache needs a matching go version to work, so append that to the name
GO_VERSION="$(go version|awk '{ print $3 }'|sed 's/go//g')"

# Make sure we never error, this is always best-effort only
exit_gracefully() {
  if [ $? -ne 0 ]; then
    echodate "Encountered error when trying to download gocache"
  fi
  exit 0
}
trap exit_gracefully EXIT

source $(dirname $0)/../lib.sh

if [ -z "${GOCACHE_MINIO_ADDRESS:-}" ]; then
  echodate "env var GOCACHE_MINIO_ADDRESS unset, can not download gocache"
  exit 0
fi

GOCACHE="$(go env GOCACHE)"
# Make sure it actually exists
mkdir -p "${GOCACHE}"

export CACHE_VERSION="${PULL_BASE_SHA:-}"

# Periodics just use their head ref
if [[ -z "${CACHE_VERSION}" ]]; then
  CACHE_VERSION="$(git rev-parse HEAD)"
fi

if [ -z "${PULL_NUMBER:-}" ]; then
  # Special case: This is called in a Postubmit. Go one revision back,
  # as there can't be a cache for the current revision
  CACHE_VERSION="$(git rev-parse ${CACHE_VERSION}~1)"
fi

ARCHIVE_NAME="${CACHE_VERSION}-${GO_VERSION}.tar"
URL="${GOCACHE_MINIO_ADDRESS}/${ARCHIVE_NAME}"

# Do not go through the retry loop when there is nothing
if ! curl --head --silent --fail "${URL}" > /dev/null; then
  echodate "Remote has no gocache ${ARCHIVE_NAME}, exiting"
  exit 0
fi

echodate "Downloading and extracting gocache"
TEST_NAME="Download and extract gocache"
# Passing the Headers as space-separated literals doesn't seem to work
# in conjunction with the retry func, so we just put them in a file instead
echo 'Content-Type: application/octet-stream' > /tmp/headers
retry 5 curl --fail -H @/tmp/headers "${URL}" | tar -C $GOCACHE -xf -

echodate "Successfully fetched gocache into $GOCACHE"
