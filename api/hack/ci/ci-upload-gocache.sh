#!/usr/bin/env bash

set -euo pipefail

# Required for signal propagation to work so
# the cleanup trap gets executed when the script
# receives a SIGINT
set -o monitor

source $(dirname $0)/../lib.sh

if [ -z ${GOCACHE_MINIO_ADDRESS:-} ]; then
  echodate "Fatal: env var GOCACHE_MINIO_ADDRESS unset"
  exit 1
fi

GOCACHE_DIR=$(mktemp -d)
export GOCACHE="${GOCACHE_DIR}"
export GIT_HEAD_HASH="$(git rev-parse HEAD|tr -d '\n')"

export CGO_ENABLED=0
cd $(dirname $0)/../..

echodate "Building binaries"
TEST_NAME="Build Kubermatic"
retry 2 make build
(
  TEST_NAME="Building Nodeport proxy"
  cd cmd/nodeport-proxy
  retry 2 make build
)
(
  TEST_NAME="Building kubeletdnat controller"
  cd cmd/kubeletdnat-controller
  retry 2 make build
)

echodate "Building tests"
TEST_NAME="Build tests"

for edition in ee ce; do
  retry 2 go test -tags "cloud $edition" ./... -run nope
  retry 2 go test -tags "create $edition" ./... -run nope
  retry 2 go test -tags "e2e $edition" ./... -run nope
  retry 2 go test -tags "integration $edition" ./... -run nope
done

echodate "Creating gocache archive"
TEST_NAME="Creating gocache archive"
ARCHIVE_FILE=/tmp/${GIT_HEAD_HASH}.tar
# No compression because that needs quite a bit of CPU
retry 2 tar -C $GOCACHE -cf $ARCHIVE_FILE .

echodate "Uploading gocache archive"
TEST_NAME="Uploading gocache archive"
# The gocache needs a matching go version to work, so append that to the name
GO_VERSION="$(go version|awk '{ print $3 }'|sed 's/go//g')"
# Passing the Headers as space-separated literals doesn't seem to work
# in conjunction with the retry func, so we just put them in a file instead
echo 'Content-Type: application/octet-stream' > /tmp/headers
retry 2 curl --fail \
  -T ${ARCHIVE_FILE} \
  -H @/tmp/headers \
  ${GOCACHE_MINIO_ADDRESS}/${GIT_HEAD_HASH}-${GO_VERSION}.tar
