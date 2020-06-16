#!/bin/bash

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

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
swagger -apiPackage="github.com/kubermatic/kubermatic/api"  \
  -mainApiFile="github.com/kubermatic/kubermatic/api/cmd/kubermatic-api/main.go" \
  -format="swagger" \
  -output="$(go env GOPATH)/src/github.com/kubermatic/kubermatic/api/handler/swagger/"
