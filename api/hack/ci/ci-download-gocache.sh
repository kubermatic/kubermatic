#!/usr/bin/env bash

set -euo pipefail

# Required for signal propagation to work so
# the cleanup trap gets executed when the script
# receives a SIGINT
set -o monitor

# Make sure we never error, this is always best-effort only
exit_gracefully() { exit 0; }
trap exit_gracefully EXIT

source $(dirname $0)/../lib.sh

export GOCACHE_MINIO_ADDRESS='http://minio.gocache.svc.cluster.local.:9000/gocache'
if [ -z ${GOCACHE_MINIO_ADDRESS:-} ]; then
  echodate "env var GOCACHE_MINIO_ADDRESS unset, can not download gocache"
  exit 0
fi

GOCACHE=$(go env GOCACHE)
# Make sure it actually exists
mkdir -p $GOCACHE

if ls -1qA $GOCACHE | grep -q .; then
  echodate "gocache at $GOCACHE is not empty, omitting download of gocache"
  exit 0
fi

CACHE_VERSION="${PULL_BASE_SHA}"
if [ -z $PULL_NUMBER ]; then
  # Special case: This is called in a Postubmit. Go one revision back,
  # as there can't be a cache for the current revision
  CACHE_VERSION=$(git rev-parse ${CACHE_VERSION}~1)
fi
# Hardcoded for testing
CACHE_VERSION=6af928d8093bbf096986170f1496fb134f7db843

TEST_NAME="Download and extract gocache"
# Passing the Headers as space-separated literals doesn't seem to work
# in conjunction with the retry func, so we just put them in a file instead
echo 'Content-Type: application/octet-stream' > /tmp/headers
retry 5 curl --fail \
    --progress-bar \
    -H @/tmp/headers \
    ${GOCACHE_MINIO_ADDRESS}/${CACHE_VERSION}.tar \
    |tar -C $GOCACHE -xvf -

echodate "Successfully fetched gocache"
