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

### Generates the KKP API Swagger spec and client. The generated client is then
### used in the api-e2e tests and published into https://github.com/kubermatic/go-kubermatic

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

CONTAINERIZE_IMAGE=golang:1.17.1 containerize ./hack/gen-api-client.sh

cd cmd/kubermatic-api/
SWAGGER_FILE="swagger.json"
TMP_SWAGGER="${SWAGGER_FILE}.tmp"

function cleanup() {
  rm $TMP_SWAGGER
}
trap cleanup EXIT SIGINT SIGTERM

go run github.com/go-swagger/go-swagger/cmd/swagger generate spec \
  --tags=ee \
  --scan-models \
  -o ${TMP_SWAGGER}

rm -r ../../pkg/test/e2e/utils/apiclient/
mkdir -p ../../pkg/test/e2e/utils/apiclient/

go run github.com/go-swagger/go-swagger/cmd/swagger generate client \
  -q \
  --skip-validation \
  -f ${TMP_SWAGGER} \
  -t ../../pkg/test/e2e/utils/apiclient/
