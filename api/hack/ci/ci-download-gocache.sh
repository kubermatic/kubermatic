#!/usr/bin/env bash

set -euo pipefail

# Required for signal propagation to work so
# the cleanup trap gets executed when the script
# receives a SIGINT
set -o monitor

# Make sure we never error, this is always best-effort only
exit_gracefully() { exit 0; }
trap exit_gracefully EXIT

source ./api/hack/lib.sh

if [ -z $GOCACHE_MINIO_ADDRESS ]; then
  echodate "env var GOCACHE_MINIO_ADDRESS unset, can not download gocache"
  exit 0
fi

GOCACHE=$(go env GOCACHE)
# Make sure it actually exists
mkdir -p $GOCACHE

if ls -1qA ./somedir/ | grep -q .; then
  echodate "gocache at $GOCACHE is not empty, omitting download of gocache"
  exit 0
fi

CACHE_VERSION="${PULL_BASE_SHA}"
if [ -z $PULL_NUMBER ]; then
  # Special case: This is called in a Postubmit. Go one revision back,
  # as there can't be a cache for the current revision
  CACHE_VERSION=$(git rev-parse ${CACHE_VERSION}~1|tr -d '\n')
fi

TEST_NAME="Download and extract gocache"
retry 5 curl --fail
    --progress-bar\
    -H "Content-Type: binary/octet-stream" \
    ${GOCACHE_MINIO_ADDRESS}/${CACHE_VERSION}.tar \
    |tar -C $GOCACHE -xvf -
