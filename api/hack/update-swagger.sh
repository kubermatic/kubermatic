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

API_DIR="$(realpath .)"
SWAGGER_FILE="swagger.json"

echodate "Installing go-swagger"
cd vendor/github.com/go-swagger/go-swagger/cmd/swagger
go install

echodate "Generating swagger spec"
cd "${API_DIR}/cmd/kubermatic-api/"
swagger generate spec --scan-models -o ${SWAGGER_FILE}
