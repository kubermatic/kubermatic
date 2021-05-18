#!/usr/bin/env bash

# Copyright 2021 The Kubermatic Kubernetes Platform contributors.
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

### Releases a new quay.io/kubermatic/alertmanager-authorization-server Docker image; must be
### run manually whenever the alertmanager-authorization-server is updated.

set -euo pipefail

cd $(dirname $0)/..

REPOSITORY=quay.io/kubermatic/alertmanager-authorization-server
TAG=0.1.0

GOOS=linux GOARCH=amd64 make alertmanager-authorization-server

mkdir --parents cmd/alertmanager-authorization-server/_build
mv _build/alertmanager-authorization-server cmd/alertmanager-authorization-server/_build/
cd cmd/alertmanager-authorization-server/

docker build -t "$REPOSITORY:$TAG" .
docker push "$REPOSITORY:$TAG"
