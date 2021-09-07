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

cd $(dirname $0)/..
source hack/lib.sh

function cleanup() {
  rm -f $TMP_SWAGGER

  if [[ -n "${TMP_DIR:-}" ]]; then
    rm -rf "${TMP_DIR}"
  fi
}
trap cleanup EXIT SIGINT SIGTERM

SWAGGER_FILE="swagger.json"
TMP_SWAGGER="${SWAGGER_FILE}.tmp"

cd cmd/kubermatic-api/
go run github.com/go-swagger/go-swagger/cmd/swagger generate spec --tags=ee --scan-models -o ${TMP_SWAGGER}
# The parameters order in the generated swagger spec json file is not
# deterministic, sort in order to avoid flake results.
# The sorting here is applied only to first level arrays, nested arrays are not
# sorted.
curr="$(jq --argfile f ${SWAGGER_FILE} -n '($f | (..  | arrays) |= sort)')"
exp="$(jq --argfile f ${TMP_SWAGGER} -n '($f | (..  | arrays) |= sort)')"
diff -Naup <(echo "${curr}") <(echo "${exp}")
go run github.com/go-swagger/go-swagger/cmd/swagger validate ${SWAGGER_FILE}
