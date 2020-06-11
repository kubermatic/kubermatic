#!/usr/bin/env bash

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

if [ -z ${GOCACHE_MINIO_ADDRESS:-} ]; then
  echodate "env var GOCACHE_MINIO_ADDRESS unset, can not download gocache"
  exit 0
fi

GOCACHE=$(go env GOCACHE)
# Make sure it actually exists
mkdir -p $GOCACHE

# Periodics just use their head ref
if [[ -z "${PULL_BASE_SHA:-}" ]]; then
  export CACHE_VERSION="$(git rev-parse HEAD)"
fi
CACHE_VERSION="${CACHE_VERSION:-${PULL_BASE_SHA}}"
if [ -z ${PULL_NUMBER:-} ]; then
  # Special case: This is called in a Postubmit. Go one revision back,
  # as there can't be a cache for the current revision
  CACHE_VERSION=$(git rev-parse ${CACHE_VERSION}~1)
fi

ARCHIVE_NAME="${CACHE_VERSION}-${GO_VERSION}.tar"

# Do not go through the retry loop when there is nothing
if curl --head ${GOCACHE_MINIO_ADDRESS}/${ARCHIVE_NAME}|grep -q 404; then
	echodate "Remote has no gocache ${ARCHIVE_NAME}, exitting"
	exit 0
fi

echodate "Downloading and extracting gocache"
TEST_NAME="Download and extract gocache"
# Passing the Headers as space-separated literals doesn't seem to work
# in conjunction with the retry func, so we just put them in a file instead
echo 'Content-Type: application/octet-stream' > /tmp/headers
retry 5 curl --fail \
    -H @/tmp/headers \
    ${GOCACHE_MINIO_ADDRESS}/${ARCHIVE_NAME} \
    |tar -C $GOCACHE -xf -

echodate "Successfully fetched gocache into $GOCACHE"
