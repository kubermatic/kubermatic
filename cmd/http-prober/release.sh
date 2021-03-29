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

ver=v0.3.2

set -euox pipefail

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags="-w -s" k8c.io/kubermatic/v2/cmd/http-prober

docker build --no-cache --pull -t quay.io/kubermatic/http-prober:$ver .
docker push quay.io/kubermatic/http-prober:$ver

rm http-prober
