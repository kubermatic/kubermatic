#!/usr/bin/env bash

# Copyright 2022 The Kubermatic Kubernetes Platform contributors.
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

cd $(dirname $0)/..
source hack/lib.sh

KUBERMATIC_EDITION="${KUBERMATIC_EDITION:-ce}"

TEST_FLAGS=""
if [ -n "${PROW_JOB_ID:-}" ]; then
  # Go creates a testlog if caching is enabled. As our unit
  # tests render addons _very_ often, logging all filesystem I/O
  # dramatically slows down the tests; disabling caching fixes it.
  # Also, since it's CI, we don't lose anything by not caching in
  # a temporary Pod.
  # Note that setting GOCACHE=off is not supported since Go 1.12:
  # https://github.com/golang/go/issues/29378#issuecomment-449383809
  TEST_FLAGS="-v -count=1"
fi

CGO_ENABLED=1 go_test unit_tests \
  -tags "unit,${KUBERMATIC_EDITION}" $TEST_FLAGS -race ./pkg/... ./cmd/... ./codegen/...
