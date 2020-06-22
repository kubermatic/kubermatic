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

function cleanup() {
  rm -f $TMP_SWAGGER

  if [[ -n "${TMP_DIR:-}" ]]; then
    rm -rf "${TMP_DIR}"
  fi
}
trap cleanup EXIT SIGINT SIGTERM

API_DIR="$(go env GOPATH)/src/github.com/kubermatic/kubermatic/api"
SWAGGER_FILE="swagger.json"
TMP_SWAGGER="${SWAGGER_FILE}.tmp"

# install swagger to temp dir
TMP_DIR=$(mktemp -d)
mkdir -p "${TMP_DIR}/bin"

cd ${API_DIR}/vendor/github.com/go-swagger/go-swagger/cmd/swagger
env "GOBIN=${TMP_DIR}/bin" go install
export PATH="${TMP_DIR}/bin:${PATH}"

cd ${API_DIR}/cmd/kubermatic-api/
swagger generate spec --scan-models -o ${TMP_SWAGGER}
diff -Naup ${SWAGGER_FILE} ${TMP_SWAGGER}

swagger validate ${SWAGGER_FILE}
