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

CONTAINERIZE_IMAGE=quay.io/kubermatic/build:go-1.18-node-16-10 containerize ./hack/update-swagger.sh

# For some reason, since go-swagger 0.30.0, GOFLAGS with `-trimpath` causes
# Swagger to ignore/forget/don't see half of the necessary types and completely
# mangles the generated spec.
# After multiple days of debugging we simply gave up and ensure that GOFLAGS
# is not set for generating/verifying the Swagger spec.
export GOFLAGS=

echodate "Generating swagger spec"
cd cmd/kubermatic-api/
go run github.com/go-swagger/go-swagger/cmd/swagger generate spec --tags=ee --scan-models -o swagger.json
echodate "Completed."
