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

### Releases a new quay.io/kubermatic/s3-exporter Docker image; must be
### run manually whenever the s3-exporter is updated.

set -euo pipefail

cd $(dirname $0)/..

export REGISTRY=quay.io/kubermatic/s3-exporter
export TAG=v0.4

GOOS=linux GOARCH=amd64 make s3-exporter

mv _build/s3-exporter cmd/s3-exporter/
cd cmd/s3-exporter/

docker build -t $REGISTRY:$TAG .
docker push $REGISTRY:$TAG
