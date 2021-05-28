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

cd $(dirname $0)

REPOSITORY=quay.io/kubermatic/openvpn
VERSION=2.5.2-r0

docker build --build-arg OPENVPN_VERSION=${VERSION} --no-cache --pull -t "${REPOSITORY}:v${VERSION}" .
docker push "${REPOSITORY}:v${VERSION}"
