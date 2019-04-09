#!/usr/bin/env bash

set -euo pipefail
# Required for signal propagation to work so
# the cleanup trap gets executed when the script
# receives a SIGINT
set -o monitor

cd $(dirname $0)

export UPGRADE_TEST_BASE_HASH=${UPGRADE_TEST_BASE_HASH:-$(git rev-parse master)}

./ci-run-minimal-conformance-tests.sh
